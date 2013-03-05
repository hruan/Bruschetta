package rt

import (
	"bruschetta/data/netflix"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"strconv"
	"time"
)

type (
	searchResponse struct {
		Count  int     `json:"total"`
		Movies []Movie `json:"movies"`
	}

	Movie struct {
		Id        string            `json:"id"`
		Title     string            `json:"title"`
		Released  map[string]string `json:"release_dates"`
		//Directors map[string]string `json:"abridged_directors"`
		//Cast      map[string]string `json:"abridged_cast"`
		Consensus string            `json:"critics_consensus"`
		Rating    Rating            `json:"ratings"`
		Links     map[string]string `json:"links"`
	}

	Rating struct {
		CriticsStr  string `json:"critics_rating"`
		Critics     int    `json:"critics_score"`
		AudienceStr string `json:"audience_rating"`
		Audience    int    `json:"audience_score"`
	}

	SearchError struct {
		id string
	}

	rtCredentials struct {
		ApiKey string
	}
)

const rateLimit = 9

var (
	apiKey rtCredentials
	bucket chan time.Time
)

func (e *SearchError) Error() string {
	return "No match for " + e.id + " found"
}

// Get JSON-representation of Movie m
func (m *Movie) AsJson() []byte {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	if err := encoder.Encode(m); err != nil {
		log.Printf("Couldn't marshal movie as JSON: %s\n", err)
		return nil
	}

	return b.Bytes()
}

// Match m against title and year; expects title to be "hyphenified"
func (m *Movie) match(title, year string) bool {
	return m.matchTitle(title) && m.matchYear(year)
}

func (m *Movie) matchTitle(title string) bool {
	t := hyphenify(m.Title)
	//log.Printf("Comparing %s with %s\n", t, title)
	return t == title
}

func (m *Movie) matchYear(year string) bool {
	const sep = "-"

	rdate, ok := m.Released["theater"]
	if !ok || !strings.Contains(rdate, sep) {
		return false
	}

	_year := strings.Split(rdate, "-")[0]
	//log.Printf("Comparing %s with %s\n", _year, year)
	return year == _year
}

// Remove all non-alphanumerical characters and replace whitespaces with "-"
func hyphenify(s string) string {
	collapse := regexp.MustCompile(`[^\w- ]+`)
	hyphen := regexp.MustCompile(`[\s-]+`)

	collapsed := collapse.ReplaceAllString(strings.ToUpper(s), "")
	hyphened := hyphen.ReplaceAllString(collapsed, "-")
	//log.Println("Hyphenified: ", hyphened)
	return hyphened
}

/* func Search(title, year string) (*Movie, error) {
	const searchURL = "http://api.rottentomatoes.com/api/public/v1.0/movies.json"

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %s\n", err)
		return nil, err
	}

	v := url.Values{}
	v.Set("limit", "10")
	for _, w := range strings.Split(title, "-") {
		v.Add("q", w)
	}
	req.URL.RawQuery = v.Encode()

	appendKey(req.URL)

	// Wait for a token to become available before sending the request
	<-bucket
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Client request failed: %s\n")
		return nil, err
	}
	defer resp.Body.Close()

	var result searchResponse
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&result); err != nil {
		log.Printf("JSON unmarshaling failed: %s\n", err)
		return nil, err
	}

	return filter(result.Movies, title, year)
} */

func Search(id string) (*Movie, error) {
	t, err := netflix.Get(id)
	if err != nil {
		log.Printf("ID not found: %s\n", err)
		return nil, &SearchError{id: id}
	}

	v := url.Values{}
	v.Set("limit", "10")
	for _, w := range strings.Fields(t.Title) {
		v.Add("q", w)
	}

	resp, err := rtSearch(v)
	if err != nil {
		log.Printf("RT search failed: %s\n", err)
		return nil, err
	}
	defer resp.Close()

	var result searchResponse
	decoder := json.NewDecoder(resp)
	if err = decoder.Decode(&result); err != nil {
		log.Printf("JSON unmarshaling failed: %s\n", err)
		return nil, err
	}

	return filter(result.Movies, t)
}

func rtSearch(v url.Values) (io.ReadCloser, error) {
	const searchURL = "http://api.rottentomatoes.com/api/public/v1.0/movies.json"

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %s\n", err)
		return nil, err
	}

	req.URL.RawQuery = v.Encode()
	appendKey(req.URL)

	// Wait for a token to become available before sending the request
	<-bucket
	log.Printf("Sending request: %+v\n", req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Client request failed: %s\n")
		return nil, err
	}

	return resp.Body, nil
}

func filter(movies []Movie, t *netflix.Title) (match *Movie, err error) {
	ht := hyphenify(t.Title)
	year := strconv.Itoa(t.Year)
	for _, m := range movies {
		if m.match(ht, year) {
			match = new(Movie)
			*match = m
			return
		}
	}

	return nil, &SearchError{id: strconv.Itoa(t.Id)}
}

// Appends RottenTomatoes API key to an URL
func appendKey(u *url.URL) {
	if apiKey.ApiKey != "" {
		v := u.Query()
		v.Set("apikey", apiKey.ApiKey)
		u.RawQuery = v.Encode()
	}
}

func init() {
	c, err := ioutil.ReadFile("rt.json")
	if err != nil {
		log.Fatalf("rt.json missing: %s\n", err)
	}

	if err = json.Unmarshal(c, &apiKey); err != nil {
		log.Fatalf("rt.json unmarshaling failed: %s\n", err)
	}

	// Rate limit RT-requests using a token bucket
	bucket = make(chan time.Time, rateLimit)
	go func() {
		t := time.Tick(time.Second / rateLimit)
		for {
			token := <-t
			bucket <- token
		}
	}()
}
