package app

import (
	"database/sql"
	"github.com/go-gorp/gorp"
	"github.com/mattn/go-sqlite3"
	"github.com/rothso/isqool/pkg/scrape"
	"log"
)

type SqliteStorage struct {
	db    *sql.DB
	dbmap *gorp.DbMap
}

func NewSqliteStorage(file string) SqliteStorage {
	storage := SqliteStorage{}

	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Panic("Unable to connect to database: ", err)
	}
	storage.db = db

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(scrape.Isq{}, "isq").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	dbmap.AddTableWithName(scrape.Grades{}, "grades").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	dbmap.AddTableWithName(scrape.Schedule{}, "schedules").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		log.Panic("Unable to create tables: ", err)
	}
	storage.dbmap = dbmap
	return storage
}

func (s SqliteStorage) Save(vs []interface{}) error {
	tx, err := s.dbmap.Begin()
	if err != nil {
		return err
	}
	for _, v := range vs {
		err := tx.Insert(v)
		if sqliteError, ok := err.(sqlite3.Error); ok {
			if sqliteError.ExtendedCode == sqlite3.ErrConstraintUnique {
				continue // silently ignore duplicates
			}
		}
	}
	return tx.Commit()
}

func (s SqliteStorage) Close() error {
	return s.db.Close()
}
