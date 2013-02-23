package netflix

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"github.com/garyburd/go-oauth/oauth"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type catalog struct {
	XMLName xml.Name `xml:"catalog_titles" json:"-"`
	Titles  []title  `xml:"catalog_title" json:"titles"`
}

type title struct {
	Id     string  `xml:"id" json:"id"`
	Name   name    `xml:"title" json:"name"`
	Year   string  `xml:"release_year" json:"year"`
	Rating float32 `xml:"average_rating" json:"rating"`
	Boxart boxart  `xml:"box_art" json:"box_art"`
	Links  []link  `xml:"link" json:"__links"`
}

// TODO: Having Name field in Title map to title>regular,attr should suffice
// but it's not supported at the moment:
// http://code.google.com/p/go/issues/detail?id=3688
type name struct {
	Short   string `xml:"short,attr" json:"short"`
	Regular string `xml:"regular,attr" json:"regular"`
}

type boxart struct {
	Small  string `xml:"small,attr" json:"small"`
	Medium string `xml:"medium,attr" json:"medium"`
	Large  string `xml:"large,attr" json:"large"`
}

type link struct {
	URL  string `xml:"href,attr" json:"url"`
	Rel  string `xml:"rel,attr" json:"rel"`
	Name string `xml:"title,attr" json:"name"`
}

var (
	client = &oauth.Client{
		TemporaryCredentialRequestURI: "http://api-public.netflix.com/oauth/request_token",
		ResourceOwnerAuthorizationURI: "http://api-user.netflix.com/oauth/login",
		TokenRequestURI:               "http://api-public.netflix.com/oauth/access_token",
	}
	configFile string
)

func Search(term string, max int) ([]byte, error) {
	resp, err := client.Get(
		http.DefaultClient,
		nil,
		"http://api-public.netflix.com/catalog/titles",
		url.Values{
			"term":        {term},
			"max_results": {strconv.Itoa(max)},
		})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Netflix search failed: %s\n", resp.Status)
		return nil, errors.New("Netflix search failed")
	}

	content, err := unmarshal(resp.Body)
	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(content.Titles)
	if err != nil {
		return nil, err
	}

	return json, nil
}


func unmarshal(reader io.Reader) (*catalog, error) {
	var catalog catalog
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&catalog); err != nil {
		log.Printf("unmarshal: xml.Decode failed: %s\n", err)
		return nil, err
	}

	return &catalog, nil
}

func readConfig(c *oauth.Client) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal("Couldn't read config: ", err)
	}

	if err := json.Unmarshal(b, &c.Credentials); err != nil {
		log.Fatal("Couldn't unmarshal credentials: ", err)
	}
}

func init() {
	flag.StringVar(&configFile, "config", "netflix.json", "Path to config file")
	readConfig(client)
}
