package file

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	// "github.com/damoonazarpazhooh/chunker/pkg/iosecure/decryptor"

	utils "github.com/damoonazarpazhooh/chunker/pkg/utils"

	"github.com/palantir/stacktrace"
)

// PutInternal -
func (b *Storage) PutInternal(ctx context.Context, entry *Entry) error {
	var err error
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Put operation.starting to validate entry key (%s)", entry.Key)
	err = b.validatePath(entry.Key)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Put operation error. could not validate entry key (%s) ", entry.Key)
		return err
	}
	path, key := b.expandPath(entry.Key)
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Put operation.making parent tree at (%s)", path)
	err = os.MkdirAll(path, 0700)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Put operation error. Could not make the parent tree at (%s)", path)
		return err
	}
	fullPath := utils.PathJoin(path, key)
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Put operation. creating empty file for the stream at (%s)", fullPath)
	f, err := os.OpenFile(
		fullPath,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0600)
	if err != nil {
		if f != nil {
			f.Close()
		}
		err = stacktrace.Propagate(err, "[ERROR] Storage: Put operation error. Could not create empty file at (%s) ", fullPath)
		return err
	}
	if f == nil {
		err = stacktrace.NewError("[ERROR] Storage: Put operation could not successfully get a file handle ")
		return err
	}
	defer f.Close()
	reader := bytes.NewBuffer(entry.Value)
	var length int64
	// reader := ratelimitedreader.New(entry.Value, b.uploadRateLimit/b.numberOfThreads)
	if b.encryptionKey != nil {

		encReader, err := b.newEncryptor(reader)
		if err != nil {
			return err
		}
		length, err = io.CopyBuffer(f, encReader, make([]byte, HeaderSize+MaxPayloadSize+TagSize))

		// 	length, err = iosecure.EncryptIO(
		// 		f,
		// 		reader,
		// 		encryptor.WithKey(b.encryptionKey),
		// 	)
	} else {
		length, err = io.Copy(f, reader)
		if err != nil {
			return err
		}
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Put operation.IO Buffer copied (%s) bytes to file at (%s)", utils.PrettyPrintSize(length), path)
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Put operation. stating file at (%s) for confirmation", path)
	fi, err := os.Stat(fullPath)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Put operation error. could not stat file at (%s) after writing to it", path)
		return err
	}
	if fi == nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Put operation error. target file for storing bytes at path (%s) was empty after writing to it", path)
		return err
	}
	if fi.Size() == 0 {
		os.Remove(fullPath)
	}
	return nil

}

// GetInternal -
// TODO FIX ERROR propogation
func (b *Storage) GetInternal(ctx context.Context, key string) (*Entry, error) {
	var err error
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Get operation.validating key (%s) ...", key)
	err = b.validatePath(key)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Get operation error.could not validate entry key (%s) ", key)
		return nil, err

	}

	path, keyExpanded := b.expandPath(key)
	path = filepath.Join(path, keyExpanded)
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Get operation.stating file at (%s)", path)

	// If we stat it and it exists but is size zero, it may be left from some
	// previous FS error like out-of-space. No entry will ever be zero
	// length, so simply remove it and return nil.
	fi, err := os.Stat(path)
	if err == nil {
		if fi.Size() == 0 {
			b.logCh <- fmt.Sprintf("[red][WARN] Storage: Get operation.Target file (%s) exists but is size zero, it may be left from some previous FS error like out-of-space ", path)
			// Best effort, ignore errors
			os.Remove(path)
			return nil, nil
		}
	}
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Get operation.Opening file at (%s)", path)

	f, err := os.Open(path)
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		if os.IsNotExist(err) {
			b.logCh <- fmt.Sprintf("[red][WARN] Storage: Get operation.file at (%s) does not exists", path)
			return nil, nil

		}
		err = stacktrace.Propagate(err, "[ERROR] Storage: Get operation error.could not open the file at (%s) ", path)
		return nil, err
	}
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Get operation. starting to read bytes from (%s) into memory", path)
	buf := bytes.NewBuffer(nil)
	// _, err = buf.ReadFrom(f)
	// if err != nil {
	// 	err = stacktrace.Propagate(err, "[ERROR] Storage: Get operation error. could not read the bytes from the opeend file ")
	// 	return nil, err
	// }
	_, err = io.CopyBuffer(buf, f, make([]byte, MaxPayloadSize))
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Get operation error. could not decrypt and read the bytes from the opeend file ")
		return nil, err
	}
	if b.encryptionKey != nil {
		target := bytes.NewBuffer(nil)
		decReader, err := b.newDecryptor(buf)
		if err != nil {
			return nil, err
		}
		_, err = io.CopyBuffer(target, decReader, make([]byte, MaxPayloadSize))
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] Storage: Get operation error. could not decrypt and read the bytes from the opeend file ")
			return nil, err
		}
		buf = target

	}
	result := &Entry{
		Key:   key,
		Value: buf.Bytes(),
	}

	return result, nil

}

// DeleteInternal -
func (b *Storage) DeleteInternal(ctx context.Context, key string) error {
	var err error
	if key == "" {
		b.logCh <- fmt.Sprintf("[red][WARN] Storage: Delete operation. the given key was an empty string")

		return nil
	}
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Delete operation.validating key (%s) ...", key)
	err = b.validatePath(key)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Delete operation error.could not validate entry key (%s) ", key)
		return err
	}
	basePath, keyExpanded := b.expandPath(key)
	fullPath := filepath.Join(basePath, keyExpanded)
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Delete operation.deleting file at (%s)", fullPath)
	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		err = stacktrace.Propagate(err, "[ERROR] Storage: Delete operation failed to remove %q", fullPath)
		return err

	}
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: Delete operation.cleaning up logical path (%s). Removing all empty nodes, beginning with deepest one, aborting on first non-empty one, up to top-level node", key)
	err = b.cleanupPath(key)
	if err != nil {
		return err
	}
	return nil
}

// ListInternal -
func (b *Storage) ListInternal(ctx context.Context, prefix string) ([]string, error) {
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: List operation.validating path prefix (%s) ...", prefix)
	err := b.validatePath(prefix)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: List operation error.could not validate path prefix (%s) ", prefix)
		return nil, err

	}
	path := b.path
	if prefix != "" {
		path = filepath.Join(path, prefix)
	}
	b.logCh <- fmt.Sprintf("[yellow][INFO] Storage: List operation.Opening file at (%s)", path)

	// Read the directory contents
	f, err := os.Open(path)
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		if os.IsNotExist(err) {
			b.logCh <- fmt.Sprintf("[red][WARN] Storage: List operation. file at (%s) does not exist", path)
			return nil, nil
		}
		err = stacktrace.Propagate(err, "[ERROR] Storage: List operation error.Could not open file at (%s) ", path)

		return nil, err

	}
	names, err := f.Readdirnames(-1)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] Storage: List operation error. could not read and return slice of names from (%v) ", path)
		return nil, err
	}

	for i, name := range names {
		fi, err := os.Stat(filepath.Join(path, name))
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] Storage: List operation error ")

			return nil, err
		}
		if fi.IsDir() {
			names[i] = name + "/"
		} else {
			if name[0] == '_' {
				names[i] = name[1:]
			}
		}
	}

	if len(names) > 0 {
		sort.Strings(names)
	}
	return names, nil

}

// cleanupPath is used to remove all empty nodes, beginning with deepest
// one, aborting on first non-empty one, up to top-level node.
func (b *Storage) cleanupPath(path string) error {
	nodes := strings.Split(path, fmt.Sprintf("%c", os.PathSeparator))
	for i := len(nodes) - 1; i > 0; i-- {
		fullPath := filepath.Join(b.path, filepath.Join(nodes[:i]...))

		dir, err := os.Open(fullPath)
		if err != nil {
			if dir != nil {
				dir.Close()
			}
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		list, err := dir.Readdir(1)
		dir.Close()
		if err != nil && err != io.EOF {
			return err
		}

		// If we have no entries, it's an empty directory; remove it
		if err == io.EOF || list == nil || len(list) == 0 {
			err = os.Remove(fullPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func (b *Storage) expandPath(k string) (string, string) {
	path := filepath.Join(b.path, k)
	key := filepath.Base(path)
	path = filepath.Dir(path)
	// return path, "_" + key
	return path, key
}

func (b *Storage) validatePath(path string) error {
	switch {
	case strings.Contains(path, ".."):
		// ErrPathContainsParentReferences
		// this error is returned when a path contains parent references.
		return stacktrace.NewError("path cannot contain parent references")
	}
	return nil
}
