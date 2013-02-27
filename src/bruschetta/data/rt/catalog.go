package rt

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
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

var apiKey rtCredentials

func (m Movie) AsJson() []byte {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	if err := encoder.Encode(m); err != nil {
		log.Printf("Couldn't marshal movie as JSON: %s\n", err)
		return nil
	}

	return b.Bytes()
}

func Search(title, year string) ([]Movie, error) {
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
	log.Printf("Query and limit: %s", req.URL)

	appendKey(req.URL)

	log.Printf("Sending request to RT: %s\n", req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Client request failed: %s\n")
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("Response: %s\n", http.StatusText(resp.StatusCode))

	var result searchResponse
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&result); err != nil {
		log.Printf("JSON unmarshaling failed: %s\n", err)
		return nil, err
	}

	fn := func(m *Movie) bool {
		y, ok := m.Released["theater"]
		if !ok {
			y = "any"
		}
		return strings.ToUpper(m.Title) == strings.ToUpper(title) && (year == "any" || y == year)
	}
	return filter(result.Movies, fn), nil
}

func filter(movies []Movie, fn func(*Movie) bool) (filtered []Movie) {
	for _, m := range movies {
		if fn(&m) {
			filtered = append(filtered, m)
		}
	}

	return
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
}
