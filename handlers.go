package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)


func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type",  "text/html; charset=utf-8")
	count := cfg.fileserverHits.Load()

	html := fmt.Sprintf(`
	<html>
	  <body>
	    <h1>Welcome, Chirpy Admin</h1>
	    <p>Chirpy has been visited %d times!</p>
	  </body>
	</html>
	`, count)
	fmt.Fprintln(w, html)
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {

	if cfg.platform == "dev" {
		cfg.fileserverHits.Store(0)
		err := cfg.db.DeleteUsers(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not reset the Users ", err)
			return
		}
		w.Header().Set("Content-Type",  "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Hits reset to 0!\n")

	} else {
		respondWithError(w, http.StatusForbidden, "403 Forbidden", nil)
	}

}


func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}

	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}
	
	respondWithJson(w, code, errorResponse{
		Error: msg,
	})

}



func respondWithJson(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(code)
	w.Write(data)
}

func removeProfanity(body string) string {
	badWords := map[string]struct{} {
		"kerfuffle": {}, 
		"sharbert": {},
		"fornax": {},
	}
	words := strings.Split(body, " ")


	for i, word := range words {
		lowered := strings.ToLower(word)
		if _, ok := badWords[lowered]; ok {
			words[i] = "****"
		}
	}
	
	cleaned := strings.Join(words, " ")
	return cleaned
}
