package handlers_test

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
}