package main

import (
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	nodeGraphQLURL = "http://localhost:8545/graphql" // Ethereum GraphQL endpoint
	authToken      = "my-secret-token"               // Replace with your desired token
	maxBodySize    = 1 << 20                         // 1 MB request body size limit
)

// Middleware for authentication
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the Authorization header
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") || strings.TrimPrefix(token, "Bearer ") != authToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Middleware to limit the size of incoming request bodies
func limitBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		next.ServeHTTP(w, r)
	})
}

// Proxy handler to forward GraphQL requests
func graphqlProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request is a POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Forward the request to the Ethereum GraphQL endpoint
	proxyReq, err := http.NewRequest(r.Method, nodeGraphQLURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Perform the request to the Ethereum node
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to connect to Ethereum GraphQL endpoint", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers and body to the client
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	// Create an HTTP handler chain with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", graphqlProxyHandler)

	// Apply middlewares
	handler := authMiddleware(limitBodySize(mux))

	// Start the server
	log.Println("GraphQL proxy server running on http://localhost:8081/graphql")
	log.Fatal(http.ListenAndServe(":8081", handler))
}
