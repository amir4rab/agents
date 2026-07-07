package domaintypes

import "database/sql"

type Cursor string

type CursorPagination struct {
	After  *Cursor `json:"after" query:"after"`
	Before *Cursor `json:"before" query:"before"`
	First  uint8   `json:"first" query:"first"`
	Last   uint8   `json:"last" query:"last"`
}

type PageInfo struct {
	StartCursor     *Cursor `json:"startCursor"`
	EndCursor       *Cursor `json:"endCursor"`
	HasNextPage     bool    `json:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage"`
	Limit           uint8   `json:"limit"`
}

type Page[Model any] struct {
	Items []*Model `json:"items"` // List of items
	Info  PageInfo `json:"info"`  // Page info
}

type Repository[Model any, ID comparable, Q any] interface {
	Create(tx *sql.Tx, model *Model) (err error)
	Update(tx *sql.Tx, model *Model) (err error)
	Delete(tx *sql.Tx, id ID) (err error)

	Get(tx *sql.Tx, id ID) (val *Model, err error)
	Exists(tx *sql.Tx, id ID) (ok bool, err error)

	Find(tx *sql.Tx, query *Q) (page *Page[Model], err error)
}
