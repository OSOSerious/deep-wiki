package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// Post represents a blog post
type Post struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

// PostStore is an interface for storing and retrieving posts
type PostStore interface {
	GetAllPosts() ([]Post, error)
	GetPost(id string) (Post, error)
	CreatePost(post Post) (Post, error)
	UpdatePost(post Post) (Post, error)
	DeletePost(id string) error
}

// MemoryPostStore is a simple in-memory implementation of PostStore
type MemoryPostStore struct {
	posts map[string]Post
}

func (store *MemoryPostStore) GetAllPosts() ([]Post, error) {
	posts := make([]Post, 0, len(store.posts))
	for _, post := range store.posts {
		posts = append(posts, post)
	}
	return posts, nil
}

func (store *MemoryPostStore) GetPost(id string) (Post, error) {
	post, ok := store.posts[id]
	if !ok {
		return Post{}, errors.New("post not found")
	}
	return post, nil
}

func (store *MemoryPostStore) CreatePost(post Post) (Post, error) {
	post.ID = fmt.Sprintf("post-%d", len(store.posts)+1)
	store.posts[post.ID] = post
	return post, nil
}

func (store *MemoryPostStore) UpdatePost(post Post) (Post, error) {
	_, ok := store.posts[post.ID]
	if !ok {
		return Post{}, errors.New("post not found")
	}
	store.posts[post.ID] = post
	return post, nil
}

func (store *MemoryPostStore) DeletePost(id string) error {
	delete(store.posts, id)
	return nil
}

func main() {
	store := &MemoryPostStore{posts: make(map[string]Post)}

	router := mux.NewRouter()

	router.HandleFunc("/posts", getPostsHandler(store)).Methods("GET")
	router.HandleFunc("/posts", createPostHandler(store)).Methods("POST")
	router.HandleFunc("/posts/{id}", getPostHandler(store)).Methods("GET")
	router.HandleFunc("/posts/{id}", updatePostHandler(store)).Methods("PUT")
	router.HandleFunc("/posts/{id}", deletePostHandler(store)).Methods("DELETE")

	http.ListenAndServe(":8000", router)
}

func getPostsHandler(store PostStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		posts, err := store.GetAllPosts()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(posts)
	}
}

func createPostHandler(store PostStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var post Post
		err := json.NewDecoder(r.Body).Decode(&post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		post, err = store.CreatePost(post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(post)
	}
}

func getPostHandler(store PostStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		post, err := store.GetPost(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(post)
	}
}

func updatePostHandler(store PostStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		var post Post
		err := json.NewDecoder(r.Body).Decode(&post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		post.ID = id
		post, err = store.UpdatePost(post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(post)
	}
}

func deletePostHandler(store PostStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		err := store.DeletePost(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}