package db

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //import postgres

	"go-dispatcher2/config"
)

var db *sqlx.DB

func init() {
	psqlInfo := fmt.Sprintf("%s", config.Dispatcher2Conf.Database.URI)

	var err error
	db, err = ConnectDB(psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	// log.Println(Schema)
}

// ConnectDB ...
func ConnectDB(dataSourceName string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	return db, nil
}

// GetDB ...
func GetDB() *sqlx.DB {
	return db
}
