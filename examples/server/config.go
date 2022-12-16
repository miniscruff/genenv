package main

import "fmt"

//go:generate go run ../../. --config=Config --file config_gen.go --env .env.example --verbose

type Config struct {
	// Host will configure the http server for what hostname to listen on
	Host string `default:"localhost"`
	// Port will configure the HTTP port to listen on
	Port int `default:"3000"`

	// DataStore handles storing our data for key values
	DataStore *DataStoreConfig
}

func (c *Config) NewServer() (*Server, error) {
	s := &Server{}
	s.Address = fmt.Sprintf("%v:%v", c.Host, c.Port)

	dataStore, err := c.DataStore.Build()
	if err != nil {
		return s, err
	}

	s.DataStore = dataStore

	return s, nil
}

// DataStoreConfig will allow loading one of the possible data storage
// types.
type DataStoreConfig struct {
	// Used by the gen to load the proper config
	// must be named "Type", a default doc string is generated?
	// buildType specifies what type our Build method should return
	Type string `buildType:"DataStore"`

	// Will use the type docs for docs
	// Must be pointers for now
	*MemDataStoreConfig    `env:"MEM"`
	*SqliteDataStoreConfig `env:"SQLITE"`
}

// MemDataStoreConfig will configure using an in memory data store.
// This is no concurrent safe and no production ready.
type MemDataStoreConfig struct {
}

func (c *MemDataStoreConfig) NewMemDataStore() (*MemDataStore, error) {
	return &MemDataStore{}, nil
}

// SqliteDataStoreConfig will configure a sqlite database for storage.
// This is concurrent safe but not production ready
type SqliteDataStoreConfig struct {
	// Filename specifies the sqlite database file path
	Filename string `env:"FILENAME" default:"data.db"`
}

func (c *SqliteDataStoreConfig) NewSqliteDataStore() (*SqliteDataStore, error) {
	return NewSqliteDataStore(c.Filename)
}
