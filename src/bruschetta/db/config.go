package db

import(
	"database/sql"
	"flag"
	_ "github.com/bmizerany/pq"
)

var (
	dbhost, dbport, dbname, dbuser, dbpass string
)

func Open() (db *sql.DB, err error) {
	db, err = sql.Open("postgres", buildConnStr())

	return
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

func init() {
	flag.StringVar(&dbhost, "dbhost", "", "port number of database")
	flag.StringVar(&dbport, "dbport", "5432", "port number of database")
	flag.StringVar(&dbname, "dbname", "bruschetta", "name of the database")
	flag.StringVar(&dbuser, "dbuser", "", "username for database")
	flag.StringVar(&dbpass, "dbpass", "", "password for database")

	if !flag.Parsed() {
		flag.Parse()
	}
}
