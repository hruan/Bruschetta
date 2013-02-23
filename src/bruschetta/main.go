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

func defaultApiHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("Received %s %s request from %s", req.Method, req.URL, req.RemoteAddr)

	catalog, err := netflix.Search("start", 5)
	if err != nil {
		msg := fmt.Sprintf("Search failed: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, "Received: %+v\n", string(catalog))
}

func main() {
	flag.Parse()
	flag.StringVar(&staticDir, "static", "content", "Directory from which to server static files")

	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir(staticDir)))
	r.HandleFunc("/api/v1", defaultApiHandler)

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
