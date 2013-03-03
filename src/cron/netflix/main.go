package main

import (
	"bruschetta/db"
	"encoding/xml"
	"flag"
	_ "github.com/bmizerany/pq"
	"github.com/garyburd/go-oauth/oauth"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	client = &oauth.Client{
		TemporaryCredentialRequestURI: "http://api-public.netflix.com/oauth/request_token",
		ResourceOwnerAuthorizationURI: "http://api-user.netflix.com/oauth/login",
		TokenRequestURI:               "http://api-public.netflix.com/oauth/access_token",
	}
)

type (
	titleIndex struct {
		Id      string  `xml:"id"`
		Year    int     `xml:"release_year"`
		Title   title   `xml:"title"`
		Updated int     `xml:"updated"`
		Rating  float32 `xml:"average_rating"`
		Links   []link  `xml:"link"`
	}

	link struct {
		Rel string `xml:"rel,attr"`
		URL string `xml:"href,attr"`
	}

	title struct {
		Short   string `xml:"short,attr"`
		Regular string `xml:"regular,attr"`
	}
)

func (t titleIndex) pathSegments() []string {
	u, err := url.Parse(t.Id)
	if err != nil {
		return []string{}
	}

	return strings.Split(u.Path, "/")
}

func (t titleIndex) id() int {
	p := t.pathSegments()
	if len(p) > 0 {
		i, err := strconv.Atoi(p[len(p)-1])
		if err != nil {
			panic("Couldn't find id")
		}
		return i
	}
	panic("Title missing URL path")
}

func (t titleIndex) movie() bool {
	p := t.pathSegments()
	l := len(p)
	if l > 0 {
		return p[l-2] == "movies"
	}
	panic("Title missing URL path")
}

func (t titleIndex) playURL() string {
	for _, t := range t.Links {
		if t.Rel == "alternate" {
			return t.URL
		}
	}

	return ""
}

func fetchFromNetflix() io.ReadCloser {
	req, err := http.NewRequest(
		"GET",
		"http://api-public.netflix.com/catalog/titles/streaming",
		nil)
	if err != nil {
		log.Fatalf("Failed to create request: %s\n", err)
	}

	req.Header.Set("Authorization",
		client.AuthorizationHeader(
			nil,
			req.Method,
			req.URL,
			nil))
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Netflix catalog request failed: %s\n", err)
	}

	return resp.Body
}

func fetchFromFile() io.ReadCloser {
	f, err := os.Open("db/netflix_catalog.bin")
	if err != nil {
		log.Fatalf("Couldn't open Netflix catalog: %s\n", err)
	}

	return f
}

func update(r io.ReadCloser, w chan<- titleIndex) {
	defer r.Close()

	var title titleIndex
	decoder := xml.NewDecoder(r)
	for t, err := decoder.Token(); err == nil; t, err = decoder.Token() {
		switch s := t.(type) {
		case xml.StartElement:
			if s.Name.Local == "catalog_title" {
				err = decoder.DecodeElement(&title, &s)
				if err != nil {
					log.Printf("DecodeElement failed: %s\n", err)
					continue
				}
				if title.movie() {
					w <- title
				}
			}
		}
	}
}

func write(c <-chan titleIndex) {
	const stmt = `INSERT INTO titles (id, year, title, updated, rating, play_url) VALUES ($1, $2, $3, $4, $5, $6)`
	const batchSize = 10

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Caught panic: %s\n", err)
		}
	}()

	db, err := db.Open()
	if err != nil {
		log.Fatalf("Couldn't open DB connection: %s\n", err)
	}
	defer db.Close()

	st, err := db.Prepare(stmt)
	if err != nil {
		log.Fatalf("Couldn't prepare SQL statement: %s\n", err)
	}
	defer st.Close()

	for {
		t, ok := <-c
		if !ok {
			log.Println("Channel closed. Stopping goroutine.")
			return
		}

		_, err := st.Exec(t.id(), t.Year, t.Title.Regular, t.Updated, t.Rating, t.playURL())
		if err != nil {
			log.Fatalf("Exec failed: %s\n", err)
		}
	}
}

func main() {
	fetch := flag.Bool("fetch", false, "fetch=<true | false>")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	const writers = 8
	c := make(chan titleIndex, writers*2)
	for i := 0; i < writers; i++ {
		go write(c)
	}
	defer close(c)

	if *fetch {
		log.Println("Fetching catalog from Netflix")
		update(fetchFromNetflix(), c)
	} else {
		log.Println("Fetching catalog from file")
		update(fetchFromFile(), c)
	}
}
