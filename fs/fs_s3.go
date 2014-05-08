package statedb

import (
	"errors"
	// "fmt"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"path/filepath"
)

type FS_S3 struct {
	fs  *s3.Bucket
	dir string
}

func NewFS_S3(auth aws.Auth, region aws.Region, dir, name string) (*FS_S3, error) {

	if name == "" || len(name) < 3 || len(name) > 63 {
		return nil, errors.New("Bucket name is of invalid length 3 <= len(bucket_name) <= 63")
	}

	return &FS_S3{
		fs:  s3.New(auth, region).Bucket(name),
		dir: dir,
	}, nil
}

func (b *FS_S3) Init() error {

	// check if bucket already exists
	_, err := b.fs.List("", "/", "", 1)
	if err != nil {
		return b.fs.PutBucket(s3.BucketOwnerFull)
	}

	return nil
}

// S3 does not have folders, but files can use a "/" delimiter to
// emulate the concept of folders.
func (b *FS_S3) Put(path string, data []byte) error {

	return b.fs.Put(
		filepath.Join(b.dir, path),
		data, "binary/octet-stream",
		s3.BucketOwnerFull)
}

func (b *FS_S3) Get(path string) ([]byte, error) {
	return b.fs.Get(filepath.Join(b.dir, path))
}

func (b *FS_S3) Delete(path string) error {
	return b.fs.Del(filepath.Join(b.dir, path))
}

func (b *FS_S3) List(pattern string) ([]string, error) {

	resp, err := b.fs.List(filepath.Join(b.dir, pattern), "/", "", 1000)
	if err != nil {
		return nil, err
	}

	items := make([]string, 0, len(resp.Contents))
	for _, k := range resp.Contents {
		items = append(items, k.Key)
	}
	return items, err
}

func (b *FS_S3) Volume() string {
	return b.fs.Name
}

func (b *FS_S3) Dir() string {
	return b.dir
}
