package user

import (
	"database/sql"
	"errors"

	domaintypes "github.com/nilafzar/agents/services/manager/internal/model/internal/types"
)

type repository struct{}

func (repository) Create(tx *sql.Tx, model *User) (err error) {
	return errors.New("not implemented")
}

func (repository) Update(tx *sql.Tx, model *User) (err error) {
	return errors.New("not implemented")

}

func (repository) Delete(tx *sql.Tx, id int64) (err error) {
	return errors.New("not implemented")

}

func (repository) Get(tx *sql.Tx, id int64) (val *User, err error) {
	return nil, errors.New("not implemented")
}

func (repository) Exists(tx *sql.Tx, id int64) (ok bool, err error) {
	return false, errors.New("not implemented")
}

func (repository) Find(tx *sql.Tx, query *Query) (page *domaintypes.Page[User], err error) {
	return nil, errors.New("not implemented")
}

func NewRepository() Repository {
	return repository{}
}
