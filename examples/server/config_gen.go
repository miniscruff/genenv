package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var (
	ErrInvalidBuildType = errors.New("invalid build type")
	ErrKeyNotFound      = errors.New("env var key not found")
)

func NewConfig() (*Config, error) {
	var (
		err error
		c   *Config
	)

	c.Host, err = ParseStringOptional("localhost", "HOST")
	if err != nil {
		return c, err
	}

	c.Port, err = ParseIntOptional("3000", "PORT")
	if err != nil {
		return c, err
	}

	c.DataStore, err = NewDataStoreConfig("DATA_STORE")
	if err != nil {
		return c, err
	}

	return c, err
}

func NewDataStoreConfig(prefix string) (*DataStoreConfig, error) {
	var (
		err error
		c   *DataStoreConfig
	)

	c.Type, err = ParseStringRequired(prefix + "_TYPE")
	if err != nil {
		return c, err
	}

	c.MemDataStoreConfig, err = NewMemDataStoreConfig(prefix + "_MEM")
	if err != nil {
		return c, err
	}

	c.SqliteDataStoreConfig, err = NewSqliteDataStoreConfig(prefix + "_SQLITE")
	if err != nil {
		return c, err
	}

	return c, err
}

func (c *DataStoreConfig) Build() (DataStore, error) {
	switch c.Type {
	case "MEM":
		return c.NewMemDataStore()
	case "SQLITE":
		return c.NewSqliteDataStore()
	default:
		return nil, fmt.Errorf("%w: %v", ErrInvalidBuildType, c.Type)
	}
}

func NewMemDataStoreConfig(prefix string) (*MemDataStoreConfig, error) {
	var (
		err error
		c   *MemDataStoreConfig
	)

	return c, err
}

func NewSqliteDataStoreConfig(prefix string) (*SqliteDataStoreConfig, error) {
	var (
		err error
		c   *SqliteDataStoreConfig
	)

	c.Filename, err = ParseStringOptional("data.db", prefix+"_FILENAME")
	if err != nil {
		return c, err
	}

	return c, err
}

func ParseStringOptional(def, key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		v = def
	}

	return v, nil
}

func ParseIntOptional(def, key string) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		v = def
	}

	v64, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(v64), nil
}

func ParseStringRequired(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("%w: %v", ErrKeyNotFound, key)
	}

	return v, nil
}
