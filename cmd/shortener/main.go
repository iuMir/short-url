package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type URLShortener struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewURLShortener() *URLShortener {
	return &URLShortener{
		urls: make(map[string]string),
	}
}

func (us *URLShortener) generateID() string {
	for {
		b := make([]byte, 6)
		rand.Read(b)
		id := base64.URLEncoding.EncodeToString(b)[:8]

		us.mu.RLock()
		_, exists := us.urls[id]
		us.mu.RUnlock()

		if !exists {
			return id
		}
	}
}

func (us *URLShortener) CreateURL(originalURL string) string {
	us.mu.Lock()
	defer us.mu.Unlock()

	id := us.generateID()
	us.urls[id] = originalURL
	return id
}

func (us *URLShortener) GetURL(id string) (string, bool) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	url, exists := us.urls[id]
	return url, exists
}

func main() {
	shortener := NewURLShortener()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			originalURL := make([]byte, r.ContentLength)
			if _, err := r.Body.Read(originalURL); err != nil && err.Error() != "EOF" {
				http.Error(w, "Invalid input", http.StatusBadRequest)
				return
			}

			id := shortener.CreateURL(string(originalURL))
			shortURL := fmt.Sprintf("http://localhost:8080/%s", id)

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(shortURL))

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := r.PathValue("id")
		originalURL, exists := shortener.GetURL(id)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
