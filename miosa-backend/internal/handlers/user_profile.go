package handlers

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
}