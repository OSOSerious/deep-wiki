package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var users = map[string]string{}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	users[user.Username] = string(hash)
	w.WriteHeader(http.StatusCreated)
}

func loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, ok := users[user.Username]
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Password))
	if err != nil {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		hash, ok := users[username]
		if !ok {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err != nil {
			http.Error(w, "invalid password", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("protected resource"))
}

func main() {
	http.HandleFunc("/users", createUserHandler)
	http.HandleFunc("/login", loginUserHandler)
	http.HandleFunc("/protected", authenticate(protectedHandler))

	http.ListenAndServe(":8080", nil)
}