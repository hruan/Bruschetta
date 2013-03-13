package rt

import (
	"bruschetta/netflix"
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
func (m *Movie) match(title string, year int) bool {
	return m.matchTitle(title) && m.matchYear(year)
}

func (m *Movie) matchTitle(title string) bool {
	t := hyphenify(m.Title)
	//log.Printf("Comparing %s with %s\n", t, title)
	return strings.Contains(t, title)
}

func (m *Movie) matchYear(year int) bool {
	const sep = "-"
	const grace = 2;

	rdate, ok := m.Released["theater"]
	if !ok || !strings.Contains(rdate, sep) {
		return false
	}

	_year, err := strconv.Atoi(strings.Split(rdate, "-")[0])
	if err != nil {
		log.Printf("Couldn't find release year: %s\n", err)
		return false
	}
	//log.Printf("Comparing %s with %s\n", _year, year)
	return _year >= year-grace && _year <= year+grace
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

// Search RT for a movie using a Netflix id. Returns review data for the first
// movie with matching year and title, if any.
func Search(id string) (*Movie, error) {
	t, err := netflix.Get(id)
	if err != nil {
		log.Printf("ID not found: %s\n", err)
		return nil, &SearchError{id: id}
	}

	resp, err := rtSearch(t)
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

func rtSearch(t *netflix.Title) (io.ReadCloser, error) {
	const searchURL = "http://api.rottentomatoes.com/api/public/v1.0/movies.json"

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %s\n", err)
		return nil, err
	}

	// Build RT request URL
	appendKey(req.URL)
	v := url.Values{}
	v.Set("limit", "10")
	req.URL.RawQuery += "&" + v.Encode()
	req.URL.RawQuery += "&q=" + escapeQuery(t.Title)

	// Wait for a token to become available before sending the request
	<-bucket
	//log.Printf("Sending request: %s\n", req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Client request failed: %s\n")
		return nil, err
	}

	return resp.Body, nil
}

// Join s with '+', removing some undesired characters
func escapeQuery(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}

	filter := regexp.MustCompile(`[^\w-.~]+`)
	words := strings.Fields(s)
	var filtered []string
	for _, w := range words {
		filtered = append(filtered, filter.ReplaceAllString(w, ""))
	}
	return strings.Join(filtered, "+")
}

func filter(movies []Movie, t *netflix.Title) (match *Movie, err error) {
	ht := hyphenify(t.Title)
	for _, m := range movies {
		if m.match(ht, t.Year) {
			match = new(Movie)
			*match = m
			return
		}
	}

	return nil, &SearchError{id: t.Id}
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
