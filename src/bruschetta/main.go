package main

import (
	"flag"
	"fmt"
	"bruschetta/data/netflix"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
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

	fmt.Fprintf(w, "Received:\n")
	for i, v := range catalog.Titles {
		fmt.Fprintf(w, "[%2v]: %+v\n", i, v)
	}
}


func main() {
	flag.Parse()
	flag.StringVar(&staticDir, "static", "content", "Directory from which to server static files")

	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir(staticDir)))
	r.HandleFunc("/api/1", defaultApiHandler)

	p := strconv.Itoa(port)
	log.Print("Listening on port ", p)
	err := http.ListenAndServe(":" + p, r)
	if err != nil {
		log.Fatal("Couldn't start server: ", err)
	}
}
