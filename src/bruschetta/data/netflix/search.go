package netflix

import (
	"bruschetta/db"
	"encoding/json"
	_ "github.com/bmizerany/pq"
	"database/sql"
	"log"
)

type (
	Title struct {
		Id	int	`json:"id"`
		Title	string	`json:"title"`
		Year	int	`json:"year"`
		URL	string	`json:"url"`
		Rating	float32	`json:"rating"`
	}
)

func (t *Title) AsJson() []byte {
	json, err := json.Marshal(t)
	if err != nil {
		log.Printf("JSON marshaling failed: %s\n", err)
		return []byte{}
	}

	return json
}

func Search(title string, year int) (titles []Title, err error) {
	db, err := db.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var rows *sql.Rows
	// TODO: Escape NUL, \, ', ", %, _, [, and ]
	if year >= 0 {
		query := "SELECT id, title, year, play_url, rating FROM titles WHERE title LIKE '%$1%' AND year = $2"
		rows, err = db.Query(query, title, year)
	} else {
		s := `%` + title + `%`
		rows, err = db.Query(`SELECT id, title, year, play_url, rating FROM titles WHERE title ILIKE $1`, s)
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

