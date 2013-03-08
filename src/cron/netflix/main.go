package main

import (
	"bruschetta/db"
	"encoding/json"
	"encoding/xml"
	_ "github.com/bmizerany/pq"
	"github.com/garyburd/go-oauth/oauth"
	"io"
	"io/ioutil"
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

const (
	credPath = "netflix_credentials.json"
	dbPath = "db/netflix.db"
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
		Rel      string  `xml:"rel,attr"`
		URL      string  `xml:"href,attr"`
		Synopsis string  `xml:"synopsis"`
		BoxArt   *boxArt `xml:"box_art"`
	}

	title struct {
		Short   string `xml:"short,attr"`
		Regular string `xml:"regular,attr"`
	}

	boxArt struct {
		Links []link `xml:"link"`
	}
)

func (t *titleIndex) pathSegments() []string {
	u, err := url.Parse(t.Id)
	if err != nil {
		return []string{}
	}

	return strings.Split(u.Path, "/")
}

func (t *titleIndex) id() int {
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

func (t *titleIndex) movie() bool {
	p := t.pathSegments()
	l := len(p)
	if l > 0 {
		return p[l-2] == "movies"
	}
	panic("Title missing URL path")
}

func (t *titleIndex) playURL() string {
	for _, l := range t.Links {
		if l.Rel == "alternate" {
			return l.URL
		}
	}

	return ""
}

func (t *titleIndex) synopsis() string {
	for _, l := range t.Links {
		if l.Rel == `http://schemas.netflix.com/catalog/titles/synopsis` {
			return l.Synopsis
		}
	}

	return ""
}

func (t *titleIndex) boxArt() string {
	for _, l := range t.Links {
		if l.BoxArt == nil {
			return ""
		}

		for _, al := range l.BoxArt.Links {
			if al.Rel == `http://schemas.netflix.com/catalog/titles/box_art/197pix_w` {
				return al.URL
			}
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
	f, err := os.Open(dbPath)
	if err != nil {
		log.Fatalf("Couldn't open Netflix catalog: %s\n", err)
	}

	return f
}

func update(r io.ReadCloser, w chan<- titleIndex) {
	defer r.Close()

	decoder := xml.NewDecoder(r)
	for t, err := decoder.Token(); err == nil; t, err = decoder.Token() {
		switch s := t.(type) {
		case xml.StartElement:
			if s.Name.Local == "catalog_title" {
				// Unmarshal to a new var each iteration as it append slices
				// giving us a slice with _all_ links from all decoded titles
				// http://golang.org/pkg/encoding/xml/#Unmarshal
				var title titleIndex
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
	const stmt = `INSERT INTO titles (id, year, title, updated, rating, play_url, synopsis, box_art) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
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

		_, err := st.Exec(t.id(), t.Year, t.Title.Regular, t.Updated, t.Rating, t.playURL(), t.synopsis(), t.boxArt())
		if err != nil {
			log.Fatalf("Exec failed: %s\n", err)
		}
	}
}

func readCredentials(fetch bool) bool {
	if !fetch {
		return true
	}

	b, err := ioutil.ReadFile(credPath)
	if err != nil {
		log.Println(err)
		return false
	}

	err = json.Unmarshal(b, &client.Credentials)
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	fetch := false
	if env := os.ExpandEnv("$CNFETCH"); env != "" {
		fetch = (env != "false" && env != "0")
	}

	if ok := readCredentials(fetch); !ok {
		log.Fatalln("Couldn't read Netflix credentials")
	}

	const writers = 4
	c := make(chan titleIndex, writers*2)
	for i := 0; i < writers; i++ {
		go write(c)
	}
	defer close(c)

	if fetch {
		log.Println("Fetching catalog from Netflix")
		update(fetchFromNetflix(), c)
	} else {
		log.Println("Fetching catalog from file")
		update(fetchFromFile(), c)
	}
}
