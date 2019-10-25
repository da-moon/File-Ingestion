package filewrapper

import (
	"log"
	"os"
	"time"

	"github.com/damoonazarpazhooh/File-Ingestion/pkg/utils"
	"github.com/mitchellh/hashstructure"
	"github.com/palantir/stacktrace"
)

var fileModeMask = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky

// File ...
type File struct {
	Root string `json:"root,omitempty" mapstructure:"root,omitempty"`
	Path string `json:"path,omitempty" mapstructure:"path,omitempty"`
	Size int64  `json:"size,omitempty" mapstructure:"size,omitempty"`
	Time int64  `json:"time,omitempty" mapstructure:"time,omitempty"`
	Mode int64  `json:"mode,omitempty" mapstructure:"mode,omitempty"`
	Hash uint64 `json:"hash,omitempty" mapstructure:"hash,omitempty"`
}

// New ...
func New(root, path string, size int64, time int64, mode uint32) *File {
	if len(path) > 0 && path[len(path)-1] != '/' && (mode&uint32(os.ModeDir)) != 0 {
		path += "/"
	}
	result := &File{
		Path: path,
		Size: size,
		Time: time,
		Mode: int64(mode),
	}
	if !result.IsDir() {
		target := utils.PathJoin(root, path)
		file, _ := os.Open(target)
		defer file.Close()
		hash, _ := hashstructure.Hash(file, nil)
		result.Hash = hash
	}
	return result

}

// RestoreMetadata ...
func (f *File) RestoreMetadata(fullPath string) bool {

	stat, err := os.Lstat(fullPath)
	fileInfo := &stat
	if err != nil {
		err = stacktrace.Propagate(
			err,
			"Failed to retrieve the file info",
		)
		log.Fatal(err)
		return false
	}
	if (*fileInfo).Mode()&fileModeMask != f.GetPermissions() {
		err := os.Chmod(fullPath, f.GetPermissions())
		if err != nil {

			err = stacktrace.Propagate(
				err,
				"Failed to set the file permissions",
			)
			log.Fatal(err)
			return false
		}
	}

	if (*fileInfo).ModTime().Unix() != f.Time {
		modifiedTime := time.Unix(f.Time, 0)
		err := os.Chtimes(fullPath, modifiedTime, modifiedTime)
		if err != nil {
			err = stacktrace.Propagate(
				err,
				"Failed to set the modification time",
			)
			log.Fatal(err)
			return false
		}
	}
	return true
}

// CreateFileFromFileInfo ...
func CreateFileFromFileInfo(fileInfo os.FileInfo, root, directory string) *File {
	// path := directory + fileInfo.Name()
	path := directory
	mode := fileInfo.Mode()

	if mode&os.ModeDir != 0 && mode&os.ModeSymlink != 0 {
		mode ^= os.ModeDir
	}

	if path[len(path)-1] != '/' && mode&os.ModeDir != 0 {
		path += "/"
	}

	result := &File{
		Root: root,
		Path: path,
		Size: fileInfo.Size(),
		Time: fileInfo.ModTime().Unix(),
		Mode: int64(mode),
	}
	if !result.IsDir() {
		target := utils.PathJoin(root, path)
		file, _ := os.Open(target)
		defer file.Close()
		// buf := new(bytes.Buffer)
		// buf.ReadFrom(file)
		// hash, _ := hashstructure.Hash(buf.Bytes(), nil)
		hash, _ := hashstructure.Hash(file, nil)
		result.Hash = hash
	}
	return result
}

// IsFile ...
func (f *File) IsFile() bool {
	return f.Mode&int64(os.ModeType) == 0
}

// IsDir ...
func (f *File) IsDir() bool {
	return f.Mode&int64(os.ModeDir) != 0
}

// GetPermissions ...
func (f *File) GetPermissions() os.FileMode {
	return os.FileMode(f.Mode) & fileModeMask
}

// IsSameAs ...
func (f *File) IsSameAs(other *File) bool {
	return f.Size == other.Size && f.Time <= other.Time+1 && f.Time >= other.Time-1
}

// IsSameAsFileInfo ...
func (f *File) IsSameAsFileInfo(other os.FileInfo) bool {
	time := other.ModTime().Unix()
	return f.Size == other.Size() && f.Time <= time+1 && f.Time >= time-1
}

// Files ...
type Files []os.FileInfo

// Len ...
func (f Files) Len() int { return len(f) }

// Swap ...
func (f Files) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// Less ...
func (f Files) Less(i, j int) bool {

	left := f[i]
	right := f[j]

	if left.IsDir() && left.Mode()&os.ModeSymlink == 0 {
		if right.IsDir() && right.Mode()&os.ModeSymlink == 0 {
			return left.Name() < right.Name()
		}
		return false
	}
	if right.IsDir() && right.Mode()&os.ModeSymlink == 0 {
		return true
	}
	return left.Name() < right.Name()
}

// ByName ...
type ByName []*File

// Len ...
func (b ByName) Len() int { return len(b) }

// Swap ...
func (b ByName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less ...
func (b ByName) Less(i, j int) bool {
	return b[i].compare(b[j]) < 0
}
func (f *File) compare(right *File) int {
	path1 := f.Path
	path2 := right.Path
	p := 0
	for ; p < len(path1) && p < len(path2); p++ {
		if path1[p] != path2[p] {
			break
		}
	}
	var c1, c2 byte
	if p < len(path1) {
		c1 = path1[p]
	}
	if p < len(path2) {
		c2 = path2[p]
	}
	c3 := c1
	for i := p; c3 != '/' && i < len(path1); i++ {
		c3 = path1[i]
	}
	c4 := c2
	for i := p; c4 != '/' && i < len(path2); i++ {
		c4 = path2[i]
	}
	if c3 == '/' {
		if c4 == '/' {
			if c1 == '/' {
				return -1
			} else if c2 == '/' {
				return 1
			} else {
				return int(c1) - int(c2)
			}
		} else {
			return 1
		}
	} else {
		if c4 == '/' {
			return -1
		}
		return int(c1) - int(c2)
	}
}
