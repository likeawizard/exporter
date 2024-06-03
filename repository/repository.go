package repository

import (
	"context"
	"database/sql"
	"fmt"
)

//go:generate go run github.com/likeawizard/exporter --name=repository --output=repository_export.go
const connStr = "user=postgres dbname=postgres password=postgres sslmode=disable"

var (
	// errNotFound is returned when the requested resource is not found
	errNotFound = fmt.Errorf("not found")
	// errInvalidName is returned when the name is not valid
	errInvalidName = fmt.Errorf("invalid name")
)

type (
	repository struct {
		conn *sql.Conn
	}

	updateQuery struct {
		name string
	}

	insertQuery struct {
		name string
	}

	Entity struct {
		ID int
	}
)

func getConnection() (*sql.Conn, error) {
	return nil, nil
}

func NewRepository(conn *sql.Conn) *repository {
	return &repository{conn: conn}
}

func (r repository) get(ctx context.Context, id int) (*Entity, error) {
	if id == 0 {
		return nil, errNotFound
	}

	return &Entity{}, nil
}

func (r *repository) list() ([]Entity, error) {
	return []Entity{}, nil
}

func (r *repository) create(q insertQuery) (str string, err error) {
	if q.name == "" {
		return "", errInvalidName
	}

	return q.name, nil
}

func (r *repository) update(id, partentID int, q updateQuery) (string, error) {
	if id == 0 {
		return "", errNotFound
	}
	if q.name == "" {
		return "", errInvalidName

	}

	return q.name, nil
}

func (r *repository) delete(id int) error {
	if id == 0 {
		return errNotFound
	}

	return nil
}
