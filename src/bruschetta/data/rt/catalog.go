package rt

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
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

	rtCredentials struct {
		ApiKey string
	}
)

const rateLimit = 9

var (
	apiKey rtCredentials
	bucket chan time.Time
)

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
	return hyphenify(m.Title) == title
}

func (m *Movie) matchYear(year string) bool {
	const sep = "-"

	rdate, ok := m.Released["theater"]
	if !ok || !strings.Contains(rdate, sep) {
		return false
	}

	_year := strings.Split(rdate, "-")[0]
	return year == _year
}

// Remove all non-alphanumerical characters and replace whitespaces with "-"
func hyphenify(s string) string {
	fn := func(r rune) rune {
		if r != ' ' && !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z')) {
			return '_'
		}
		return r
	}

	escaped := strings.Map(fn, strings.ToUpper(s))
	cleaned := strings.Replace(escaped, "_", "", -1)
	f := strings.Fields(cleaned)
	str := strings.Join(f, "-")

	// log.Println("Hyphenified:", str)
	return str
}

func Search(title, year string) (*Movie, error) {
	const searchURL = "http://api.rottentomatoes.com/api/public/v1.0/movies.json"

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %s\n", err)
		return nil, err
	}

	v := url.Values{}
	v.Set("limit", "10")
	for _, w := range strings.Fields(title) {
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

	return filter(result.Movies, hyphenify(title), year)
}

func filter(movies []Movie, title, year string) (match *Movie, err error) {
	for _, m := range movies {
		if m.match(title, year) {
			match = new(Movie)
			*match = m
			return
		}
	}

	return nil, errors.New("Not found")
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
