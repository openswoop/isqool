package database

import (
	"database/sql"
	"errors"
	"github.com/go-gorp/gorp/v3"
	"github.com/mattn/go-sqlite3"
	"github.com/openswoop/isqool/pkg/scrape"
	"log"
)

type Sqlite struct {
	db    *sql.DB
	dbmap *gorp.DbMap
}

func NewSqlite(file string) Sqlite {
	sqlite := Sqlite{}

	// Initialize the database connection
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Panic("Unable to connect to database: ", err)
	}
	sqlite.db = db

	// Initialize the database mapping, creating the tables if it's our first run
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(scrape.CourseIsq{}, "isq").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	dbmap.AddTableWithName(scrape.CourseGrades{}, "grades").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	dbmap.AddTableWithName(scrape.CourseSchedule{}, "schedules").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		log.Panic("Unable to create tables: ", err)
	}
	sqlite.dbmap = dbmap

	return sqlite
}

func (s Sqlite) SaveIsqs(isqs []scrape.CourseIsq) error {
	var insertData = make([]interface{}, len(isqs))
	for i := range isqs {
		insertData = append(insertData, &isqs[i])
	}
	return s.save(insertData)
}

func (s Sqlite) SaveGrades(grades []scrape.CourseGrades) error {
	var insertData = make([]interface{}, len(grades))
	for i := range grades {
		insertData = append(insertData, &grades[i])
	}
	return s.save(insertData)
}

func (s Sqlite) SaveSchedules(schedules []scrape.CourseSchedule) error {
	var insertData = make([]interface{}, len(schedules))
	for i := range schedules {
		insertData = append(insertData, &schedules[i])
	}
	return s.save(insertData)
}

func (s Sqlite) save(rows []interface{}) error {
	tx, err := s.dbmap.Begin()
	if err != nil {
		return err
	}
	for _, row := range rows {
		err := tx.Insert(row)
		var sqliteError sqlite3.Error
		if errors.As(err, &sqliteError) {
			if errors.Is(sqliteError.ExtendedCode, sqlite3.ErrConstraintUnique) {
				continue // silently ignore duplicates
			}
		}
	}
	return tx.Commit()
}

func (s Sqlite) Close() error {
	return s.db.Close()
}
