package main

import (
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	const port = "8080"
	const filePathRoot = "."
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir(filePathRoot))
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(fs)))


	mux.HandleFunc("GET /healthz", handlerReadiness)
	mux.HandleFunc("GET /metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /reset", apiCfg.handlerReset)

	server := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	log.Printf("Serving files from %s on port: %s\n", filePathRoot, port)
	log.Fatal(server.ListenAndServe())

}



