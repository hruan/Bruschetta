package main

import (
	"bruschetta/data/netflix"
	"bruschetta/data/rt"
	"bytes"
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const port = 8888

var staticDir string

func defaultApiHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(error); ok {
				log.Printf("defaultApiHandler panic: %s", e)
			} else {
				log.Print("defaultApiHandler panic")
			}

			log.Printf("Stack trace:\n%s", debug.Stack())
			http.Error(w, "Search is temporarily unavailable.", http.StatusInternalServerError)
		}
	}()

	log.Printf("Received %s %s request from %s", r.Method, r.URL, r.RemoteAddr)

	q := r.URL.Query()
	vars := mux.Vars(r)
	switch vars["action"] {
	case "search":
		terms, ok := q["term"]
		if !ok {
			http.Error(w, "Query 'term' is required", http.StatusBadRequest)
			return
		}

		t := strings.TrimSpace(terms[0])
		if t == "" {
			http.Error(w, "term requires an argument", http.StatusBadRequest)
			return
		}

		catalog, err := netflix.Search(t, -1)
		if err != nil {
			http.Error(w, "Search is temporarily unavailable", http.StatusInternalServerError)
		}

		w.Header().Add("Content-Type", "application/json")
		j, err := json.Marshal(catalog)
		if err != nil {
			log.Printf("JSON marshaling failed: %s\n", err)
			http.Error(w, "Search is temporarily unavailable", http.StatusInternalServerError)
		}
		w.Write(j)
	case "info":
		name, ok := q["name"]
		if !ok {
			http.Error(w, "Query 'name' is required", http.StatusBadRequest)
			return
		}

		n := strings.TrimSpace(name[0])
		if n == "" {
			http.Error(w, "name requires an argument", http.StatusBadRequest)
			return
		}

		year, ok := q["year"]
		var y string
		if !ok {
			y = "any"
		} else {
			y = year[0]
		}

		movies, err := rt.Search(n, y)
		if err != nil {
			http.Error(w, "Temporarily unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		var b bytes.Buffer
		for _, m := range movies {
			if _, err := b.Write(m.AsJson()); err != nil {
				log.Printf("Write to buffer failed: %s\n", err)
				http.Error(w, "Temporarily unavailable", http.StatusInternalServerError)
				return
			}
		}

		_, err = w.Write(b.Bytes())
	default:
		http.Error(w, "No such action", http.StatusNotFound)
	}
} */

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
	movies, err := rt.Search(title, year)
	if err != nil {
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	var b bytes.Buffer
	for _, m := range movies {
		if _, err := b.Write(m.AsJson()); err != nil {
			log.Printf("Write to buffer failed: %s\n", err)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
	}

	_, err = w.Write(b.Bytes())
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
	api.Path("/reviews/{year:^[\\d]{4}$}/{title:([:word:]|[- ])+}").Methods("GET").HandlerFunc(reviewHandler)

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
