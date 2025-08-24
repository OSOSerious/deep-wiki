package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

// FileContent represents the content to save
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// IDEClient handles communication with the IDE server
type IDEClient struct {
	BaseURL string
}

// SaveFile saves content to a file via IDE API
func (c *IDEClient) SaveFile(path string, content string) error {
	payload := FileContent{
		Path:    path,
		Content: content,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/api/ide/file", c.BaseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error: %s", string(body))
	}

	return nil
}

func main() {
	ideClient := &IDEClient{BaseURL: "http://localhost:8085"}
	rootPath := "/Users/ososerious/OSA/miosa-backend/internal"
	
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║     MIOSA Multi-Agent IDE Integration Demonstration      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// Simulate different agents creating files
	agents := []struct {
		Name  string
		Color string
	}{
		{"🤖 Analysis Agent", "\033[36m"},    // Cyan
		{"🏗️  Architecture Agent", "\033[33m"}, // Yellow
		{"💻 Development Agent", "\033[32m"},   // Green
		{"🧪 Testing Agent", "\033[35m"},       // Magenta
		{"📊 Quality Agent", "\033[34m"},       // Blue
	}
	
	// Example workflow: Creating a new API endpoint
	fmt.Println("📋 Task: Create a new user profile API endpoint")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(1 * time.Second)
	
	// Analysis Agent
	fmt.Printf("\n%s%s\033[0m analyzing requirements...\n", agents[0].Color, agents[0].Name)
	time.Sleep(500 * time.Millisecond)
	
	analysisDoc := `# User Profile API Analysis

## Requirements
- GET /api/profile/{userId} - Retrieve user profile
- PUT /api/profile/{userId} - Update user profile
- Authentication required via JWT

## Data Model
- userId: string (UUID)
- username: string
- email: string
- bio: string (optional)
- avatar: string (URL, optional)
- createdAt: timestamp
- updatedAt: timestamp`

	err := ideClient.SaveFile(filepath.Join(rootPath, "docs/user-profile-analysis.md"), analysisDoc)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("  ✓ Created analysis document: docs/user-profile-analysis.md")
	}
	
	// Architecture Agent
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\n%s%s\033[0m designing service architecture...\n", agents[1].Color, agents[1].Name)
	time.Sleep(500 * time.Millisecond)
	
	modelCode := `package models

import "time"

// UserProfile represents a user's profile data
type UserProfile struct {
	UserID    string    ` + "`json:\"userId\"`" + `
	Username  string    ` + "`json:\"username\"`" + `
	Email     string    ` + "`json:\"email\"`" + `
	Bio       string    ` + "`json:\"bio,omitempty\"`" + `
	Avatar    string    ` + "`json:\"avatar,omitempty\"`" + `
	CreatedAt time.Time ` + "`json:\"createdAt\"`" + `
	UpdatedAt time.Time ` + "`json:\"updatedAt\"`" + `
}`

	err = ideClient.SaveFile(filepath.Join(rootPath, "models/user_profile.go"), modelCode)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("  ✓ Created model: models/user_profile.go")
	}
	
	// Development Agent
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\n%s%s\033[0m implementing API handlers...\n", agents[2].Color, agents[2].Name)
	time.Sleep(500 * time.Millisecond)
	
	handlerCode := `package handlers

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"miosa-backend/internal/models"
)

// GetUserProfile handles GET /api/profile/{userId}
func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	// TODO: Fetch from database
	profile := models.UserProfile{
		UserID:   userID,
		Username: "demo_user",
		Email:    "user@example.com",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// UpdateUserProfile handles PUT /api/profile/{userId}
func UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	
	var profile models.UserProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	profile.UserID = userID
	// TODO: Update in database
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}`

	err = ideClient.SaveFile(filepath.Join(rootPath, "handlers/user_profile.go"), handlerCode)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("  ✓ Created handler: handlers/user_profile.go")
	}
	
	// Testing Agent
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\n%s%s\033[0m creating test suite...\n", agents[3].Color, agents[3].Name)
	time.Sleep(500 * time.Millisecond)
	
	testCode := `package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/gorilla/mux"
	"miosa-backend/internal/handlers"
)

func TestGetUserProfile(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/profile/123", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/profile/{userId}", handlers.GetUserProfile)
	router.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}`

	err = ideClient.SaveFile(filepath.Join(rootPath, "handlers/user_profile_test.go"), testCode)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("  ✓ Created tests: handlers/user_profile_test.go")
	}
	
	// Quality Agent
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("\n%s%s\033[0m adding monitoring & validation...\n", agents[4].Color, agents[4].Name)
	time.Sleep(500 * time.Millisecond)
	
	validationCode := `package middleware

import (
	"net/http"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	apiRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "profile_api_requests_total",
			Help: "Total profile API requests",
		},
		[]string{"method", "endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(apiRequests)
}

// ValidateAuth checks JWT token
func ValidateAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			apiRequests.WithLabelValues(r.Method, r.URL.Path, "401").Inc()
			return
		}
		
		// TODO: Validate JWT token
		apiRequests.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
		next(w, r)
	}
}`

	err = ideClient.SaveFile(filepath.Join(rootPath, "middleware/auth2.go"), validationCode)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println("  ✓ Created middleware: middleware/auth2.go")
	}
	
	// Summary
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Multi-Agent Collaboration Complete!")
	fmt.Println("\n📁 Files created by agents:")
	fmt.Println("   • docs/user-profile-analysis.md")
	fmt.Println("   • models/user_profile.go")
	fmt.Println("   • handlers/user_profile.go")
	fmt.Println("   • handlers/user_profile_test.go")
	fmt.Println("   • middleware/auth2.go")
	fmt.Println("\n🎯 Next steps:")
	fmt.Println("   1. Open IDE at http://localhost:3000/ide")
	fmt.Println("   2. Navigate to the created files in the file tree")
	fmt.Println("   3. Review and modify the generated code")
	fmt.Println("\n💡 This demonstrates how MIOSA agents collaborate to:")
	fmt.Println("   • Analyze requirements")
	fmt.Println("   • Design architecture")
	fmt.Println("   • Implement features")
	fmt.Println("   • Create tests")
	fmt.Println("   • Add monitoring")
}