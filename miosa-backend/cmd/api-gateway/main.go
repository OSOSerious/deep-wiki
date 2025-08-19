package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
)

type healthResponse struct {
    Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    resp := healthResponse{Status: "ok"}
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Println("/health checked")
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)

    addr := ":8080"
    if fromEnv := os.Getenv("PORT"); fromEnv != "" {
        addr = ":" + fromEnv
    }
    log.Printf("MIOSA API Gateway listening on %s", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Fatalf("server error: %v", err)
    }
}
