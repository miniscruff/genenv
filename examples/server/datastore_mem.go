package main

import "context"

type MemDataStore struct {
}

func NewMemDataStore() (*MemDataStore, error) {
	return &MemDataStore{}, nil
}

func (s *MemDataStore) Read(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (s *MemDataStore) Write(ctx context.Context, key, value string) error {
	return nil
}

func (s *MemDataStore) Delete(ctx context.Context, key string) (string, error) {
	return "", nil
}
