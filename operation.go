package chunker

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/damoonazarpazhooh/chunker/internal/jsonutil"
	"github.com/damoonazarpazhooh/chunker/pkg/file"
	"github.com/damoonazarpazhooh/chunker/pkg/filewrapper"
	"github.com/damoonazarpazhooh/chunker/pkg/section"
	"github.com/damoonazarpazhooh/chunker/pkg/utils"
	"github.com/mitchellh/colorstring"
	"github.com/mitchellh/hashstructure"
	"github.com/palantir/stacktrace"
)

// Snapshot ...
func (s *Multipart) Snapshot(ctx context.Context, tag string) error {
	if s.logOps {
		start := time.Now()
		defer func() {
			duration := fmt.Sprintf("[DEBUG] Split Snapshot operation took (%v) to complete", time.Now().Sub(start))
			log.Println(duration)
		}()
	}

	md, err := s.NewMetadata(tag)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Failed to extract metadata for (%s)", s.root)
		return err
	}
	openedFiles := md.Entities
	for _, v := range openedFiles {
		if !v.IsFile() || v.Size == 0 {
			continue
		}
		fullPath := utils.PathJoin(s.root, v.Path)
		colorstring.Printf("[cyan][Snapshot] : opening (%s)\n", fullPath)
		osfile, err := os.OpenFile(fullPath, os.O_RDONLY, 0)

		// could not open file ... possibly is a dir...
		if err != nil {
			v.Size = 0
			continue
		}

		s.wg.Add(1)
		defer osfile.Close()
		go s.split(ctx, v, osfile, tag, md)
		// s.split(ctx, v, osfile, tag, md)
	}
	s.wg.Wait()
	md.EndTime = time.Now().Unix()
	mdJSON, err := jsonutil.EncodeJSONWithIndentation(md)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] could not encode snapshot metadata as json")
		return err
	}
	payload := &file.Entry{
		Key:   utils.PathJoin(s.rootMetaName, tag),
		Value: mdJSON,
	}
	err = s.disk.Put(ctx, payload)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] : Splitter failed to store snapshot (%s) metadata on disk\n", tag)
		return err
	}
	return nil
}

func (s *Multipart) split(ctx context.Context, fw *filewrapper.File, osfile *os.File, tag string, metadata *SnapshotMetadata) error {
	defer s.wg.Done()
	// defer osfile.Close()
	fileSize := fw.Size
	filePath := fw.Path
	cs := s.chunkSize
	nchunks := int((fileSize + cs - 1) / cs)
	rem := fileSize % cs

	i := 0
	for {
		if i >= nchunks {
			break
		}
		offset := int64(i) * cs
		size := cs
		index := i
		if rem != 0 {
			if i == nchunks-1 {
				size = rem

			}
		}
		c := section.New(
			offset,
			size,
			index,
			osfile,
			nil,
		)
		// if s.encryptionKey != nil {
		// 	c.WithEncryption(s.encryptionKey)
		// }

		chunkPathDir := utils.PathJoin(
			s.rootChunksDir,
			tag,
			filePath,
			fmt.Sprintf("%d", index),
		)
		value, _ := c.Data()
		hash := c.Hash
		payload := &file.Entry{
			Key:   utils.PathJoin(chunkPathDir, hash),
			Value: value,
		}
		md, err := jsonutil.EncodeJSONWithIndentation(c)
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] could not encode chunk (%s) metadata", chunkPathDir)
		}
		mdPayload := &file.Entry{
			Key:   utils.PathJoin(chunkPathDir, ".metadata"),
			Value: md,
		}
		s.wg.Add(1)
		s.permitpool.Acquire()
		go func() {
			defer s.permitpool.Release()
			defer s.wg.Done()
			var err error
			err = s.disk.Put(ctx, payload)
			if err != nil {
				err = stacktrace.Propagate(err, "[ERROR] : Splitter failed to store chunk (%s) on disk\n", hash)
				log.Fatal(err)
				return
			}
			err = s.disk.Put(ctx, mdPayload)
			if err != nil {
				err = stacktrace.Propagate(err, "[ERROR] : Splitter failed to store chunk (%s) metadata on disk\n", hash)
				log.Fatal(err)
				return
			}
			s.stateLock.Lock()
			if metadata.ChunkMap[fw.Hash] == nil {
				metadata.ChunkMap[fw.Hash] = make([]*section.Section, 0)
			}
			metadata.ChunkMap[fw.Hash] = append(metadata.ChunkMap[fw.Hash], c)
			s.stateLock.Unlock()
		}()
		i++
	}
	return nil
}

// Restore ...
func (s *Multipart) Restore(ctx context.Context, restoreRoot, tag string) error {
	if s.logOps {
		start := time.Now()
		defer func() {
			duration := fmt.Sprintf("[DEBUG] Restore Snapshot operation took (%v) to complete", time.Now().Sub(start))
			log.Println(duration)
		}()
	}
	result, err := s.disk.Get(ctx, utils.PathJoin(s.rootMetaName, tag))
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Failed to retrieve metadata for (%s)", tag)
		return err
	}
	md := &SnapshotMetadata{}
	err = jsonutil.DecodeJSON(result.Value, md)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Failed to decode metadata of (%s)", tag)
		return err
	}
	snapshotFiles := md.Entities
	for _, v := range snapshotFiles {
		if !v.IsFile() || v.Size == 0 {
			continue
		}
		fullPath := utils.PathJoin(restoreRoot, tag, v.Path)
		dirs := prefixes(fullPath)
		for _, vv := range dirs {
			os.MkdirAll(utils.PathJoin(s.root, vv), 0700)
		}
		destination, err := os.OpenFile(
			utils.PathJoin(s.root, fullPath),
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			0600)
		if err != nil {
			if destination != nil {
				destination.Close()
			}
			err = stacktrace.Propagate(err, "[ERROR] Merge operation error. Could not create empty file at (%s) ", fullPath)
			return err
		}
		if destination == nil {
			err = stacktrace.NewError("[ERROR] Merge operation could not successfully get a file handle ")
			return err
		}
		defer destination.Close()
		s.wg.Add(1)
		// s.permitpool.Acquire()
		go s.merge(ctx, v, destination, md)
		// go func(file *filewrapper.File, metadata *SnapshotMetadata) {
		// 	err := s.merge(ctx, file, destination, metadata)
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// }(v, md)
	}
	s.wg.Wait()
	for _, v := range snapshotFiles {
		if !v.IsFile() || v.Size == 0 {
			continue
		}
		fullPath := utils.PathJoin(restoreRoot, tag, v.Path)
		file, err := os.OpenFile(utils.PathJoin(s.root, fullPath), os.O_RDONLY, 0)
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] could not open merged file for hash verification")
			log.Fatal(err)
		}
		hash, _ := hashstructure.Hash(file, nil)

		// en, err := s.disk.Get(ctx, fullPath)
		// if en == nil || en.Value == nil {
		// 	err = stacktrace.Propagate(err, "[ERROR] could not open restored file")
		// }
		// hash, _ := hashstructure.Hash(en.Value, nil)
		colorstring.Printf("[green]Original Hash : %v\n", v.Hash)
		colorstring.Printf("[green]Original Size : %v\n", utils.PrettyPrintSize(v.Size))
		colorstring.Printf("[cyan]restored Hash : %v\n", hash)
		st, _ := file.Stat()
		colorstring.Printf("[cyan]restored Size : %v\n", utils.PrettyPrintSize(st.Size()))

	}
	return nil
}
func (s *Multipart) merge(ctx context.Context, fw *filewrapper.File, destination *os.File, metadata *SnapshotMetadata) error {
	// defer func() {
	defer s.wg.Done()
	// }()
	tag := metadata.Tag
	for _, v := range metadata.ChunkMap[fw.Hash] {
		s.wg.Add(1)
		s.permitpool.Acquire()
		go func(sec *section.Section) {
			defer s.permitpool.Release()
			defer s.wg.Done()
			targetChunkPath := utils.PathJoin(
				s.rootChunksDir,
				tag,
				fw.Path,
				fmt.Sprintf("%d", sec.Number),
				sec.Hash,
			)
			chunkEntity, err := s.disk.Get(ctx, targetChunkPath)
			if err != nil {
				err = stacktrace.Propagate(err, "[ERROR] restoring snapshot (%s) failed due to error in retrieving chunk #%d (%s)", tag, sec.Number, sec.Hash)
				log.Fatal(err)
				return
			}
			if chunkEntity == nil {
				err = stacktrace.NewError("[ERROR] restoring snapshot (%s) failed since retrieved chunk entity #%d (%s) was empty", tag, sec.Number, sec.Hash)
				log.Fatal(err)
				return
			}
			if chunkEntity.Value == nil {
				err = stacktrace.NewError("[ERROR] restoring snapshot (%s) failed since retrieved chunk entity #%d (%s) underlying bytes was empty", tag, sec.Number, sec.Hash)
				log.Fatal(err)
				return
			}
			c := section.New(
				sec.Start,
				sec.Size,
				sec.Number,
				nil,
				destination,
			)
			// if s.encryptionKey != nil {
			// 	c.WithEncryption(s.encryptionKey)
			// }

			_, err = c.Merge(chunkEntity.Value)
			if err != nil {
				err = stacktrace.Propagate(err, "[ERROR] restoring snapshot (%s) failed due not being able to copy chunk #%d (%s)to target file", tag, sec.Number, sec.Hash)
				log.Fatal(err)
				return
			}
		}(v)
	}
	return nil
}
func prefixes(s string) []string {
	components := strings.Split(s, "/")
	result := []string{}
	for i := 1; i < len(components); i++ {
		result = append(result, strings.Join(components[:i], "/"))
	}
	return result
}
