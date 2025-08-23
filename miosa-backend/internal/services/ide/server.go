// Package ide provides a main server to run the IDE service
package ide

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

// Server represents the IDE server
type Server struct {
	IDEService *IDEService
	Port       string
}

// NewServer creates a new IDE server
func NewServer(rootPath, port string) *Server {
	return &Server{
		IDEService: NewIDEService(rootPath),
		Port:       port,
	}
}

// Start starts the IDE server
func (s *Server) Start() error {
	r := mux.NewRouter()
	
	// Register IDE API routes
	s.IDEService.RegisterRoutes(r)
	
	// Serve Svelte app build files
	svelteDir := filepath.Join(filepath.Dir(s.IDEService.RootPath), "miosa-web", "build")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(svelteDir)))
	
	log.Printf("IDE Server starting on port %s", s.Port)
	log.Printf("Serving files from: %s", s.IDEService.RootPath)
	log.Printf("Web interface at: http://localhost:%s", s.Port)
	
	return http.ListenAndServe(":"+s.Port, r)
}
