package chunker

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/damoonazarpazhooh/File-Ingestion/internal/permitpool"
	"github.com/damoonazarpazhooh/File-Ingestion/pkg/file"
	"github.com/damoonazarpazhooh/File-Ingestion/pkg/filewrapper"
	"github.com/damoonazarpazhooh/File-Ingestion/pkg/section"
	"github.com/damoonazarpazhooh/File-Ingestion/pkg/utils"
	"github.com/kardianos/osext"
	"github.com/mitchellh/colorstring"
	"github.com/palantir/stacktrace"
)

// Multipart ...
type Multipart struct {
	stateLock sync.RWMutex
	logOps    bool
	logCh     chan string
	// ----------------------
	root                   string
	rootMetaName           string
	rootChunksDir          string
	encryptionKey          string
	encryptionHeaderString string
	chunkSize              int64
	gzipCompressionLevel   int
	wg                     sync.WaitGroup
	disk                   *file.Storage
	permitpool             permitpool.PermitPool
}

// New ...
func New(opts ...Option) *Multipart {
	var err error
	result := &Multipart{
		logCh:                  make(chan string),
		encryptionHeaderString: "",
	}
	for _, opt := range opts {
		opt(result)
	}

	if len(result.root) != 0 {
		result.root, err = filepath.Abs(result.root)
		if err != nil {
			err = stacktrace.Propagate(err, "[FATAL] Splitter : Error setting up root path (%s) for splitter", result.root)
			log.Fatal(err)
		}
	} else {
		selfPath, _ := osext.ExecutableFolder()
		result.root = utils.PathJoin(selfPath, "tmp")
	}
	if len(result.rootMetaName) == 0 {
		result.rootMetaName = ".metadata"
	}
	if len(result.rootChunksDir) == 0 {
		result.rootChunksDir = ".chunks"
	}

	// TODO FIX THIS
	disk := file.New(
		file.WithNumberOfThreads(1),
		file.WithPath(result.root),
		file.WithEncryption(result.encryptionKey),
	)
	if result.logOps {
		disk = file.New(
			file.WithNumberOfThreads(1),
			file.WithPath(result.root),
			file.WithEncryption(result.encryptionKey),
			file.LogOps(),
		)
	}
	err = disk.Init()
	if err != nil {
		err = stacktrace.Propagate(err, "[FATAL] Splitter : Error setting up new filesystem")
		log.Fatal(err)
	}
	result.disk = disk
	result.permitpool = permitpool.New(
		permitpool.WithPermits(1),
	)
	if result.chunkSize == 0 {
		// chunk size : 8 MiB default
		result.chunkSize = int64(8 * 1 << 20)
	}
	return result
}

// SnapshotMetadata ...
type SnapshotMetadata struct {
	Tag           string                        `json:"tag" mapstructure:"tag"`
	StartTime     int64                         `json:"start_time" mapstructure:"start_time"`
	EndTime       int64                         `json:"end_time" mapstructure:"end_time"`
	NumberOfFiles int                           `json:"number_of_files" mapstructure:"number_of_files"`
	Entities      []*filewrapper.File           `json:"entities" mapstructure:"entities"`
	ChunkMap      map[uint64][]*section.Section `json:"chunk-map" mapstructure:"chunk-map"`
}

// NewMetadata ...
func (s *Multipart) NewMetadata(tag string) (*SnapshotMetadata, error) {
	colorstring.Printf("[cyan][Snapshot] : preparing metadata for tag (%s)\n", tag)
	result := &SnapshotMetadata{
		Tag:           tag,
		StartTime:     time.Now().Unix(),
		NumberOfFiles: 0,
		Entities:      make([]*filewrapper.File, 0),
		ChunkMap:      make(map[uint64][]*section.Section),
	}
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		mode := info.Mode()
		if !mode.IsDir() {
			entity := filewrapper.New(s.root, strings.TrimPrefix(path, s.root), info.Size(), info.ModTime().Unix(), uint32(mode))
			result.NumberOfFiles++
			result.Entities = append(result.Entities, entity)
		}
		return nil
	})
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] could not generate metadata for snapshot with tag (%s)", tag)
		return nil, err
	}
	colorstring.Printf("[cyan][Snapshot] : metadata for tag (%s) was prepared successfully\n", tag)
	return result, nil
}

// NewMetadataLegacy ...
func (s *Multipart) NewMetadataLegacy(tag string) (*SnapshotMetadata, error) {
	colorstring.Printf("[cyan][Snapshot] : preparing metadata for tag (%s)\n", tag)

	entities := []*filewrapper.File{filewrapper.New(s.root, "", 0, 0, 0)}
	result := make([]*filewrapper.File, 0)
	for len(entities) > 0 {
		entity := entities[len(entities)-1]
		entities = entities[:len(entities)-1]
		if !entity.IsDir() {
			result = append(result, entity)
		}
		path := entity.Path
		entityList := &result
		colorstring.Printf("[cyan][Snapshot] : listing entities with shared prefix for (%s)\n", path)

		subdirectories, _, err := s.listEntities(
			path,
			entityList,
		)
		if err != nil {
			err = stacktrace.Propagate(err, "Failed to list subdirectory")
			log.Println(err.Error())
			continue
		}
		entities = append(entities, subdirectories...)
	}

	result = result[1:]
	// numOfFiles := 0
	// for _, v := range result {
	// 	if v.IsFile() {
	// 		numOfFiles++
	// 	}
	// }
	colorstring.Printf("[cyan][Snapshot] : metadata for tag (%s) was prepared successfully\n", tag)

	// return &SnapshotMetadata{
	// 	Tag:       tag,
	// 	StartTime: time.Now().Unix(),
	// 	// NumberOfFiles: numOfFiles,
	// 	Entities: result,
	// 	ChunkMap: make(map[uint64][]*section.Section),
	// }, nil
	return nil, nil
}

// listEntities ...
func (s *Multipart) listEntities(path string, entityList *[]*filewrapper.File) (directoryList []*filewrapper.File, skippedFiles []string, err error) {
	fullPath := utils.PathJoin(s.root, path)
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return directoryList, nil, err
	}
	normalizedPath := path
	if len(normalizedPath) > 0 && normalizedPath[len(normalizedPath)-1] != '/' {
		normalizedPath += "/"
	}

	normalizedTop := s.root
	if normalizedTop != "" && normalizedTop[len(normalizedTop)-1] != '/' {
		normalizedTop += "/"
	}

	sort.Sort(filewrapper.Files(files))
	entries := make([]*filewrapper.File, 0, 4)
	for _, f := range files {
		// skipif entity name is the same as metadata entity or chunks
		if f.Name() == s.rootMetaName || f.Name() == s.rootChunksDir {
			continue
		}
		entry := filewrapper.CreateFileFromFileInfo(f, s.root, normalizedPath)
		if f.Mode()&(os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
			skippedFiles = append(skippedFiles, entry.Path)
			continue
		}
		entries = append(entries, entry)
	}
	if path == "" {
		sort.Sort(filewrapper.ByName(entries))
	}
	for _, entry := range entries {
		if entry.IsDir() {
			directoryList = append(directoryList, entry)
		} else {
			*entityList = append(*entityList, entry)
		}
	}
	for i, j := 0, len(directoryList)-1; i < j; i, j = i+1, j-1 {
		directoryList[i], directoryList[j] = directoryList[j], directoryList[i]
	}
	return directoryList, skippedFiles, nil
}
