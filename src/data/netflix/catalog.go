package netflix

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"github.com/garyburd/go-oauth/oauth"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type Catalog struct {
	XMLName xml.Name `xml:"catalog_titles"`
	Titles  []Title  `xml:"catalog_title"`
}

type Title struct {
	Id     string  `xml:"id"`
	Name   name    `xml:"title"`
	Year   string  `xml:"release_year"`
	Rating float32 `xml:"average_rating"`
	Boxart boxart  `xml:"box_art"`
	Links  []link  `xml:"link"`
}

// TODO: Having Name field in Title map to title>regular,attr should suffice
// but it's not supported at the moment:
// http://code.google.com/p/go/issues/detail?id=3688
type name struct {
	Short   string `xml:"short,attr"`
	Regular string `xml:"regular,attr"`
}

type boxart struct {
	Small  string `xml:"small,attr"`
	Medium string `xml:"medium,attr"`
	Large  string `xml:"large,attr"`
}

type link struct {
	URL  string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Name string `xml:"title,attr"`
}

var (
	client = &oauth.Client{
		TemporaryCredentialRequestURI: "http://api-public.netflix.com/oauth/request_token",
		ResourceOwnerAuthorizationURI: "http://api-user.netflix.com/oauth/login",
		TokenRequestURI:               "http://api-public.netflix.com/oauth/access_token",
	}
	configFile string
)

func Search(term string, max int) (*Catalog, error) {
	resp, err := client.Get(http.DefaultClient,
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

	var content []byte
	if content, err = ioutil.ReadAll(resp.Body); err == nil {
		var catalog Catalog
		if err = unmarshalContent(content, &catalog); err == nil {
			return &catalog, nil
		}
	}

	return nil, err
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

func unmarshalContent(content []byte, catalog *Catalog) error {
	return xml.Unmarshal(content, catalog)
}

func init() {
	flag.StringVar(&configFile, "config", "netflix.json", "Path to config file")
	readConfig(client)
}
