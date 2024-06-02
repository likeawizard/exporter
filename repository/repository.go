package repository

import (
	"database/sql"
	"fmt"
)

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
)

func getConnection() (*sql.Conn, error) {
	return nil, nil
}

func NewRepository(conn *sql.Conn) *repository {
	return &repository{conn: conn}
}

func (r repository) get(id int) (string, error) {
	if id == 0 {
		return "", errNotFound
	}

	return "Hello, World!", nil
}

//
//func (r *repository) list() ([]string, error) {
//	return []string{"Hello", "World!"}, nil
//}

func (r *repository) create(q insertQuery) (string, error) {
	if q.name == "" {
		return "", errInvalidName
	}

	return q.name, nil
}

func (r *repository) update(id int, q updateQuery) (string, error) {
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
