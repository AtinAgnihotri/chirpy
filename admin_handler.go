package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func AdminHandler(cfg *ApiConfig) http.Handler {
	r := chi.NewRouter()

	// metrics endpoints
	r.Get("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`<html>

		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		
		</html>`, cfg.fileServerHits)))
		return
	}))

	return r
}
