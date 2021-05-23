package repository

import (
	"github.com/vlslav/web-broker/internal/pkg/model"
)

type RepoIf interface {
	New(filename string) RepoIf
	Get(key string) (string, error)
	Put(putReq *model.PutValue) error
}

type FileRepo struct {
	fileName string
}

func (fr *FileRepo) New (filename string) RepoIf{
	// todo init
	return &FileRepo{fileName: filename}
}

func NewFileRepo(fileName string) *FileRepo {
	return &FileRepo{fileName: fileName}
}

func (fr *FileRepo) Get(getReq string) (string, error) {
	// TODO: impl
	return "", nil
}

func (fr *FileRepo) Put(putReq *model.PutValue) error {
	// TODO: impl
	return nil
}
