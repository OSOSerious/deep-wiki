package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Post represents a blog post
type Post struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// PostService interface defines the CRUD operations for blog posts
type PostService interface {
	CreatePost(post *Post) error
	GetPost(id string) (*Post, error)
	UpdatePost(post *Post) error
	DeletePost(id string) error
	GetAllPosts() ([]*Post, error)
}

// PostServiceImpl implements the PostService interface
type PostServiceImpl struct {
	posts []*Post
}

// CreatePost creates a new blog post
func (p *PostServiceImpl) CreatePost(post *Post) error {
	post.ID = fmt.Sprintf("post-%d", len(p.posts)+1)
	p.posts = append(p.posts, post)
	return nil
}

// GetPost retrieves a blog post by ID
func (p *PostServiceImpl) GetPost(id string) (*Post, error) {
	for _, post := range p.posts {
		if post.ID == id {
			return post, nil
		}
	}
	return nil, fmt.Errorf("post not found")
}

// UpdatePost updates an existing blog post
func (p *PostServiceImpl) UpdatePost(post *Post) error {
	for i, existingPost := range p.posts {
		if existingPost.ID == post.ID {
			p.posts[i] = post
			return nil
		}
	}
	return fmt.Errorf("post not found")
}

// DeletePost deletes a blog post by ID
func (p *PostServiceImpl) DeletePost(id string) error {
	for i, post := range p.posts {
		if post.ID == id {
			p.posts = append(p.posts[:i], p.posts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("post not found")
}

// GetAllPosts retrieves all blog posts
func (p *PostServiceImpl) GetAllPosts() ([]*Post, error) {
	return p.posts, nil
}

// NewPostService returns a new instance of PostServiceImpl
func NewPostService() PostService {
	return &PostServiceImpl{posts: []*Post{}}
}

// PostHandler handles HTTP requests for blog posts
type PostHandler struct {
	service PostService
}

// NewPostHandler returns a new instance of PostHandler
func NewPostHandler(service PostService) *PostHandler {
	return &PostHandler{service: service}
}

// CreatePostHandler handles POST requests to create a new blog post
func (h *PostHandler) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.service.CreatePost(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// GetPostHandler handles GET requests to retrieve a blog post by ID
func (h *PostHandler) GetPostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	post, err := h.service.GetPost(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(post)
}

// UpdatePostHandler handles PUT requests to update an existing blog post
func (h *PostHandler) UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var post Post
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	post.ID = id
	err = h.service.UpdatePost(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeletePostHandler handles DELETE requests to delete a blog post by ID
func (h *PostHandler) DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	err := h.service.DeletePost(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetAllPostsHandler handles GET requests to retrieve all blog posts
func (h *PostHandler) GetAllPostsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := h.service.GetAllPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(posts)
}

func main() {
	service := NewPostService()
	handler := NewPostHandler(service)

	router := mux.NewRouter()
	router.HandleFunc("/posts", handler.CreatePostHandler).Methods("POST")
	router.HandleFunc("/posts/{id}", handler.GetPostHandler).Methods("GET")
	router.HandleFunc("/posts/{id}", handler.UpdatePostHandler).Methods("PUT")
	router.HandleFunc("/posts/{id}", handler.DeletePostHandler).Methods("DELETE")
	router.HandleFunc("/posts", handler.GetAllPostsHandler).Methods("GET")

	log.Fatal(http.ListenAndServe(":8000", router))
}