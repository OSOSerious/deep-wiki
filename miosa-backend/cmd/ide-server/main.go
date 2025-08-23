package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/sormind/OSA/miosa-backend/internal/services/ide"
)

func main() {
	var (
		port     = flag.String("port", "8080", "Port to run the IDE server on")
		rootPath = flag.String("root", ".", "Root directory to serve files from")
	)
	flag.Parse()

	// Convert to absolute path
	absPath, err := filepath.Abs(*rootPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	server := ide.NewServer(absPath, *port)
	
	log.Printf("Starting OSA IDE Server...")
	log.Printf("Root directory: %s", absPath)
	log.Printf("Server port: %s", *port)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
