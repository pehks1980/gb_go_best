package repository

import "github.com/vlslav/web-broker/internal/pkg/model"

type PgRepo struct{}

func NewPgRepo() *PgRepo {
	return &PgRepo{}
}

func (pgr *PgRepo) New (filename string) RepoIf{
	// todo init
	return &PgRepo{}
}

func (pgr *PgRepo) Get(getReq string) (string, error) {
	// TODO: impl
	return "", nil
}

func (pgr *PgRepo) Put(putReq *model.PutValue) error {
	// TODO: impl
	return nil
}
