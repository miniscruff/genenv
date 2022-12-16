package main

import "context"

// DateStore is a generic interface for storing a key value pair
type DataStore interface {
	Read(ctx context.Context, key string) (string, error)
	Write(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) (string, error)
}
