package main

import (
	"log"
	"net/http"
)

type Server struct {
	Address   string
	DataStore DataStore
}

func (s *Server) Serve() error {
	return http.ListenAndServe(s.Address, nil)
}

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	server, err := config.NewServer()
	if err != nil {
		log.Fatal(err)
	}

	// server.Serve?
	log.Println(server)
}
