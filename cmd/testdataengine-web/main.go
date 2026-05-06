package main

import (
	"log"
	"net/http"
	"os"

	"TestDataEngine/internal/webapi"
)

func main() {
	server := webapi.NewServer(webapi.ServerDeps{
		Catalog:      webapi.StaticCatalog(),
		QueryService: webapi.NewQueryService(),
		FacetService: webapi.NewFacetService(),
		StaticDir:    "ui/dist",
	})

	addr := getenv("HTTP_ADDR", ":8080")
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Routes()))
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
