package main

import (
	"database/sql"
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
	dbhost, dbport, dbname, dbuser, dbpass string
)

type (
	titleIndex struct {
		Id      string  `xml:"id"`
		Year    string  `xml:"release_year"`
		Title   title   `xml:"title"`
		Updated int     `xml:"updated"`
		Rating  float64 `xml:"average_rating"`
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
					log.Fatalf("DecodeElement failed: %s\n", err)
				}
				w <- title
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

	db, err := sql.Open("postgres", buildConnStr())
	if err != nil {
		log.Fatalf("Couldn't open connection to database: %s\n", err)
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

		_, err := st.Exec(findId(&t), t.Year, t.Title.Regular, t.Updated, t.Rating, playURL(&t))
		if err != nil {
			log.Fatalf("Exec failed: %s\n", err)
		}
	}
}

func playURL(t *titleIndex) string {
	for _, l := range t.Links {
		if l.Rel == "http://schemas.netflix.com/catalog/title/ref.tiny" {
			return l.URL
		}
	}
	panic("Found title without a play URL")
}

func findId(t *titleIndex) int {
	u, err := url.Parse(t.Id)
	if err != nil {
		panic("Unknown id type found")
	}

	p := strings.Split(u.Path, "/")
	i, err := strconv.Atoi(p[len(p)-1])
	if err != nil {
		panic("Couldn't find id")
	}
	return i
}

func buildConnStr() string {
	connStr := "dbname=" + dbname

	if dbhost != "" {
		connStr += " host=" + dbhost
	}

	if dbport != "" {
		connStr += " port=" + dbport
	}

	if dbuser != "" {
		connStr += " user=" + dbuser
	}

	if dbpass != "" {
		connStr += " password=" + dbpass
	}

	return connStr
}

func main() {
	fetch := flag.Bool("fetch", false, "fetch=<true | false>")
	flag.StringVar(&dbuser, "dbuser", "", "username for database")
	flag.StringVar(&dbpass, "dbpass", "", "password for database")
	flag.StringVar(&dbname, "dbname", "bruschetta", "name of the database")
	flag.StringVar(&dbhost, "dbhost", "", "hostname of database")
	flag.StringVar(&dbport, "dbport", "5432", "port number of database")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	const writers = 4
	c := make(chan titleIndex, writers*2)
	for i := 0; i < writers; i++ {
		go write(c)
	}

	if *fetch {
		log.Println("Fetching catalog from Netflix")
		update(fetchFromNetflix(), c)
	} else {
		log.Println("Fetching catalog from file")
		update(fetchFromFile(), c)
	}
}
