package main

import (
	"fmt"
	"log"
	"net/http"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	var server http.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	corsMux := corsMiddleware(mux)
	server.Handler = corsMux
	server.Addr = ":8080"
	// server.

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	fmt.Println("Listening on Port", server.Addr)

}

// package main

// import (
// 	"context"
// 	"log"
// 	"net/http"
// 	"os"
// 	"os/signal"
// )

// func main() {
// 	var srv http.Server

// 	idleConnsClosed := make(chan struct{})
// 	go func() {
// 		sigint := make(chan os.Signal, 1)
// 		signal.Notify(sigint, os.Interrupt)
// 		<-sigint

// 		// We received an interrupt signal, shut down.
// 		if err := srv.Shutdown(context.Background()); err != nil {
// 			// Error from closing listeners, or context timeout:
// 			log.Printf("HTTP server Shutdown: %v", err)
// 		}
// 		close(idleConnsClosed)
// 	}()

// 	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
// 		// Error starting or closing listener:
// 		log.Fatalf("HTTP server ListenAndServe: %v", err)
// 	}

// 	<-idleConnsClosed
// }
