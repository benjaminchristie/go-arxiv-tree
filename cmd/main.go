package main

import (
	"log"

	"github.com/benjaminchristie/go-arxiv-tree/internal/api"
	"github.com/benjaminchristie/go-arxiv-tree/internal/parser"
)

func main() {
	p := api.QueryRequest{
		SearchQuery: "au:\"Losey\"",
		Cat: "cs.RO",
	}
	res, err := api.Query("query", p)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%v", parser.ParseXML(res))
}
