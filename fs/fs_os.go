package statedb

import (
	// "errors"
	// "fmt"
	"io/ioutil"
	// "launchpad.net/goamz/aws"
	// "launchpad.net/goamz/s3"
	"os"
	p "path"
	"path/filepath"
	// "sort"
)

type FS_OS struct {
	Dir string
}

func NewFS_OS(dir string) (*FS_OS, error) {
	return &FS_OS{dir}, nil
}

func (fs *FS_OS) Init() error {
	return os.MkdirAll(fs.Dir, os.ModePerm)
}

func (fs *FS_OS) Put(path string, data []byte) error {

	// create directories before writing file
	// Example: path = "test/dir/file.cpt"
	// - creates dir test and test/dir and writes file
	dir := p.Dir(path)
	if err := os.MkdirAll(filepath.Join(fs.Dir, dir), os.ModePerm); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(fs.Dir, path), data, os.ModePerm)
}

func (fs *FS_OS) Get(name string) ([]byte, error) {
	f, err := os.Open(filepath.Join(fs.Dir, name))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

// Removes the named file or directory
func (fs *FS_OS) Delete(name string) error {
	return os.Remove(filepath.Join(fs.Dir, name))
}

// Returns a list of file that matches the
// shell file name pattern
func (fs *FS_OS) List(pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(fs.Dir, pattern))
}
