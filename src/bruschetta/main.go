package main

import (
	"bruschetta/data/netflix"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"time"
)

const port = 8888

var staticDir string

func defaultApiHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s %s request from %s", r.Method, r.URL, r.RemoteAddr)

	q := r.URL.Query()
	vars := mux.Vars(r)
	switch vars["action"] {
	case "search":
		term, ok := q["term"]
		if !ok {
			http.Error(w, "Query 'term' is required", http.StatusBadRequest)
		}

		max, err := strconv.Atoi(q.Get("limit"))
		if err != nil {
			max = 10
		}

		catalog, err := netflix.Search(term[0], max)
		if err != nil {
			http.Error(w, "Search is temporarily unavailable.", http.StatusInternalServerError)
		}

		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, "Received: %+v\n", string(catalog))
	default:
		http.Error(w, "No such action", http.StatusNotFound)
	}
}

func main() {
	flag.Parse()
	flag.StringVar(&staticDir, "static", "content", "Directory from which to server static files")

	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir(staticDir)))
	r.HandleFunc("/api/1/{action:[a-z]+}", defaultApiHandler)

	p := strconv.Itoa(port)
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
