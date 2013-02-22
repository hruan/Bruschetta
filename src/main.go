package main

import (
	"flag"
	"fmt"
	"data/netflix"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

const (
	port = 8888
)

func defaultHandler(w http.ResponseWriter, req *http.Request) {
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

	r := mux.NewRouter()
	r.HandleFunc("/", defaultHandler)

	p := strconv.Itoa(port)
	log.Print("Listening on port ", p)
	err := http.ListenAndServe(":" + p, r)
	if err != nil {
		log.Fatal("Couldn't start server: ", err)
	}
}
