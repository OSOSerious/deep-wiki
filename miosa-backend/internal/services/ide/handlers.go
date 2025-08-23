// Package ide provides HTTP handlers for a basic IDE service
// that can view, edit, and manage code files
package ide

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// FileInfo represents metadata about a file or directory
type FileInfo struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"modTime"`
	Language string    `json:"language,omitempty"`
}

// CodeContent represents the content of a code file
type CodeContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Lines    int    `json:"lines"`
}

// IDEService handles IDE-related operations
type IDEService struct {
	RootPath string
}

// NewIDEService creates a new IDE service
func NewIDEService(rootPath string) *IDEService {
	return &IDEService{
		RootPath: rootPath,
	}
}

// RegisterRoutes registers all IDE routes with the router
func (s *IDEService) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/ide").Subrouter()
	
	// Add CORS middleware
	api.Use(corsMiddleware)
	
	// File operations
	api.HandleFunc("/files", s.ListFiles).Methods("GET")
	api.HandleFunc("/file", s.GetFile).Methods("GET")
	api.HandleFunc("/file", s.SaveFile).Methods("POST")
	api.HandleFunc("/file", s.DeleteFile).Methods("DELETE")
	
	// Directory operations
	api.HandleFunc("/tree", s.GetFileTree).Methods("GET")
	api.HandleFunc("/search", s.SearchFiles).Methods("GET")
	
	// Code history and previous versions
	api.HandleFunc("/history", s.GetFileHistory).Methods("GET")
	api.HandleFunc("/recent", s.GetRecentFiles).Methods("GET")
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ListFiles returns files in a directory
func (s *IDEService) ListFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = s.RootPath
	}
	
	// Security check - ensure path is within root
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(s.RootPath)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read directory: %v", err), http.StatusInternalServerError)
		return
	}
	
	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		fullPath := filepath.Join(path, entry.Name())
		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    fullPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		
		if !entry.IsDir() {
			fileInfo.Language = getLanguageFromExtension(entry.Name())
		}
		
		files = append(files, fileInfo)
	}
	
	// Sort directories first, then files
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// GetFile returns the content of a specific file
func (s *IDEService) GetFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path parameter is required", http.StatusBadRequest)
		return
	}
	
	// Security check
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(s.RootPath)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}
	
	lines := strings.Count(string(content), "\n") + 1
	if len(content) == 0 {
		lines = 0
	}
	
	response := CodeContent{
		Path:     path,
		Content:  string(content),
		Language: getLanguageFromExtension(filepath.Base(path)),
		Lines:    lines,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SaveFile saves content to a file
func (s *IDEService) SaveFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Security check
	if !strings.HasPrefix(filepath.Clean(req.Path), filepath.Clean(s.RootPath)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(req.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}
	
	if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

// DeleteFile deletes a file
func (s *IDEService) DeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path parameter is required", http.StatusBadRequest)
		return
	}
	
	// Security check
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(s.RootPath)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	if err := os.Remove(path); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// GetFileTree returns a hierarchical file tree
func (s *IDEService) GetFileTree(w http.ResponseWriter, r *http.Request) {
	tree, err := s.buildFileTree(s.RootPath, 0, 3) // Max depth of 3
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build file tree: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

// FileTreeNode represents a node in the file tree
type FileTreeNode struct {
	FileInfo
	Children []FileTreeNode `json:"children,omitempty"`
}

func (s *IDEService) buildFileTree(path string, depth, maxDepth int) (FileTreeNode, error) {
	if depth > maxDepth {
		return FileTreeNode{}, nil
	}
	
	info, err := os.Stat(path)
	if err != nil {
		return FileTreeNode{}, err
	}
	
	node := FileTreeNode{
		FileInfo: FileInfo{
			Name:    filepath.Base(path),
			Path:    path,
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		},
	}
	
	if !info.IsDir() {
		node.Language = getLanguageFromExtension(filepath.Base(path))
		return node, nil
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		return node, nil // Return node without children if can't read dir
	}
	
	for _, entry := range entries {
		// Skip hidden files and common ignore patterns
		if strings.HasPrefix(entry.Name(), ".") || 
		   entry.Name() == "node_modules" || 
		   entry.Name() == "vendor" {
			continue
		}
		
		childPath := filepath.Join(path, entry.Name())
		child, err := s.buildFileTree(childPath, depth+1, maxDepth)
		if err != nil {
			continue
		}
		
		node.Children = append(node.Children, child)
	}
	
	// Sort children
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir
		}
		return node.Children[i].Name < node.Children[j].Name
	})
	
	return node, nil
}

// SearchFiles searches for files by name or content
func (s *IDEService) SearchFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("type") // "name" or "content"
	
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	if searchType == "" {
		searchType = "name"
	}
	
	var results []FileInfo
	
	err := filepath.Walk(s.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Skip common ignore patterns
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "vendor") {
			return filepath.SkipDir
		}
		
		if searchType == "name" {
			if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(query)) {
				fileInfo := FileInfo{
					Name:    info.Name(),
					Path:    path,
					IsDir:   info.IsDir(),
					Size:    info.Size(),
					ModTime: info.ModTime(),
				}
				if !info.IsDir() {
					fileInfo.Language = getLanguageFromExtension(info.Name())
				}
				results = append(results, fileInfo)
			}
		} else if searchType == "content" && !info.IsDir() {
			// Only search in text files
			if isTextFile(info.Name()) {
				content, err := os.ReadFile(path)
				if err == nil && strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
					results = append(results, FileInfo{
						Name:     info.Name(),
						Path:     path,
						IsDir:    false,
						Size:     info.Size(),
						ModTime:  info.ModTime(),
						Language: getLanguageFromExtension(info.Name()),
					})
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetFileHistory returns recent file modifications (mock implementation)
func (s *IDEService) GetFileHistory(w http.ResponseWriter, r *http.Request) {
	// This is a basic implementation - in a real IDE you'd track actual history
	var history []map[string]interface{}
	
	err := filepath.Walk(s.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		
		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		
		if isTextFile(info.Name()) {
			history = append(history, map[string]interface{}{
				"path":     path,
				"name":     info.Name(),
				"modTime":  info.ModTime(),
				"size":     info.Size(),
				"language": getLanguageFromExtension(info.Name()),
			})
		}
		
		return nil
	})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Sort by modification time (most recent first)
	sort.Slice(history, func(i, j int) bool {
		timeI := history[i]["modTime"].(time.Time)
		timeJ := history[j]["modTime"].(time.Time)
		return timeI.After(timeJ)
	})
	
	// Limit to 50 most recent
	if len(history) > 50 {
		history = history[:50]
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetRecentFiles returns recently modified files
func (s *IDEService) GetRecentFiles(w http.ResponseWriter, r *http.Request) {
	var recent []FileInfo
	
	err := filepath.Walk(s.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		
		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		
		// Only include files modified in the last 7 days
		if time.Since(info.ModTime()) < 7*24*time.Hour && isTextFile(info.Name()) {
			recent = append(recent, FileInfo{
				Name:     info.Name(),
				Path:     path,
				IsDir:    false,
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				Language: getLanguageFromExtension(info.Name()),
			})
		}
		
		return nil
	})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recent files: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Sort by modification time (most recent first)
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].ModTime.After(recent[j].ModTime)
	})
	
	// Limit to 20 most recent
	if len(recent) > 20 {
		recent = recent[:20]
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recent)
}

// Helper functions

func getLanguageFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sql":
		return "sql"
	case ".sh":
		return "bash"
	case ".dockerfile":
		return "dockerfile"
	default:
		return "text"
	}
}

func isTextFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	textExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".java": true,
		".cpp": true, ".c": true, ".cs": true, ".php": true, ".rb": true,
		".rs": true, ".swift": true, ".kt": true, ".scala": true,
		".html": true, ".css": true, ".scss": true, ".sass": true,
		".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".md": true, ".txt": true, ".sql": true, ".sh": true,
		".dockerfile": true, ".gitignore": true, ".env": true,
		".conf": true, ".ini": true, ".toml": true,
	}
	
	return textExts[ext] || ext == ""
}