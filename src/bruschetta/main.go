package main

import (
	"bruschetta/data/netflix"
	"bruschetta/data/rt"
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const port = 8888

var staticDir string

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := strings.TrimSpace(query.Get("q"))
	if q == "" {
		http.Error(w, "Query q can't be empty", http.StatusBadRequest)
		return
	}
	log.Printf("Search request: %s\n", q)

	results, err := netflix.Search(q, -1)
	if err != nil {
		http.Error(w, "Search is temporarily unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	j, err := json.Marshal(results)
	if err != nil {
		log.Printf("Failed to marshal search results as JSON: %s\n", err)
		http.Error(w, "Search is temporarily unavailable", http.StatusInternalServerError)
		return
	}
	w.Write(j)
}

func reviewHandler(w http.ResponseWriter, r *http.Request) {
	const errStr = "RT review summary is unavailable."
	vars := mux.Vars(r)

	year, yok := vars["year"]
	title, tok := vars["title"]
	if !(yok && tok) {
		http.Error(w, "year or title missing", http.StatusBadRequest)
		return
	}

	log.Printf("Looking for reviews for %s (%s)\n", title, year)
	m, err := rt.Search(title, year)
	if err != nil {
		log.Printf("RT search failed: %s\n", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(m.AsJson())
}

func main() {
	flag.Parse()
	flag.StringVar(&staticDir, "static", "content", "Directory from which to server static files")

	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	r := mux.NewRouter()
	api := r.PathPrefix("/api/1/").Subrouter()
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	// API resources; paths are relative to "/api/1/", though it must start with "/"
	api.Path("/search").Methods("GET").Queries("q", "").HandlerFunc(searchHandler)
	api.Path("/reviews/{year:\\d{4}}/{title:(\\w|[- ])+}").Methods("GET").HandlerFunc(reviewHandler)

	p := strconv.Itoa(port)
	log.Print("Serving static files from ", staticDir)
	log.Print("Listening on port ", p)
	s := &http.Server{
		Addr:         ":" + p,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal("Couldn't start server: ", err)
	}
}
