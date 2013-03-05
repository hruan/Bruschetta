package netflix

import (
	"bruschetta/db"
	"encoding/json"
	_ "github.com/bmizerany/pq"
	"database/sql"
	"log"
	"strings"
)

type (
	Title struct {
		Id	int	`json:"id"`
		Title	string	`json:"title"`
		Year	int	`json:"year"`
		URL	string	`json:"url"`
		Rating	float32	`json:"rating"`
	}

	SearchError struct {
		msg string
	}
)

func (e *SearchError) Error() string {
	return e.msg
}

var conn *sql.DB

func (t *Title) AsJson() []byte {
	json, err := json.Marshal(t)
	if err != nil {
		log.Printf("JSON marshaling failed: %s\n", err)
		return []byte{}
	}

	return json
}

func Search(title string, year int) (titles []Title, err error) {
	var rows *sql.Rows
	// TODO: Escape NUL, \, ', ", %, _, [, and ]
	if year >= 0 {
		query := `SELECT id, title, year, play_url, rating FROM titles WHERE title ILIKE $1 and year = $2`
		rows, err = conn.Query(query, title, year)
	} else {
		s := `%` + title + `%`
		rows, err = conn.Query(`SELECT id, title, year, play_url, rating FROM titles WHERE title ILIKE $1`, s)
	}

	if err != nil {
		log.Printf("Query failed: %s\n", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var title Title
		err := rows.Scan(&title.Id, &title.Title, &title.Year, &title.URL, &title.Rating)
		if err != nil {
			log.Printf("Scan failed: %s\n", err)
			continue
		}
		titles = append(titles, title)
	}
	err = rows.Err()
	if err != nil {
		log.Printf("rows iteration failed: %s\n", err)
	}

	return
}

func Get(id string) (t *Title, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &SearchError{msg: "Id can't be empty"}
	}

	t = new(Title)
	var rows *sql.Rows
	rows, err = conn.Query(`SELECT id, title, year, play_url, rating FROM titles WHERE id = $1`, id)
	if err != nil {
		log.Printf("DB query failed: %s\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&t.Id, &t.Title, &t.Year, &t.URL, &t.Rating)
		if err != nil {
			log.Printf("Scan failed: %s\n", err)
			return
		}
	}
	err = rows.Err()
	if err != nil {
		log.Printf("rows iteration failed: %s\n", err)
	}

	return t, nil
}

func init() {
	var err error
	conn, err = db.Open()
	if err != nil {
		log.Fatalf("Couldn't open DB connection: %s\n", err)
	}
}
