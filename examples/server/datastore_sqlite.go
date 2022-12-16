package main

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SqliteDataStore struct {
	db *sql.DB
}

func NewSqliteDataStore(filename string) (*SqliteDataStore, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	return &SqliteDataStore{
		db: db,
	}, nil
}

func (s *SqliteDataStore) Read(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (s *SqliteDataStore) Write(ctx context.Context, key, value string) error {
	return nil
}

func (s *SqliteDataStore) Delete(ctx context.Context, key string) (string, error) {
	return "", nil
}
