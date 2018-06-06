package main

import (
	"github.com/go-gorp/gorp"
	"database/sql"
	"github.com/mattn/go-sqlite3"
	"log"
)

type Persistable interface {
	Persist(tx Transaction) error
}

type Feature interface {
	Persist(tx Transaction, courseKey CourseKey) error
}

type Transaction interface {
	Insert(list ...interface{}) error
}

type InsertFunc func(...interface{}) error

func (f InsertFunc) Insert(list ...interface{}) error {
	return f(list...)
}

func InsertIgnoringDupes(t Transaction) Transaction {
	return InsertFunc(func(list ...interface{}) error {
		err := t.Insert(list...)
		if sqliteError, ok := err.(sqlite3.Error); ok {
			if sqliteError.ExtendedCode == sqlite3.ErrConstraintUnique {
				return nil; // silently ignore
			}
		}
		return err
	})
}

type PrimaryKey struct {
	ID uint64 `db:"id, primarykey, autoincrement" csv:"-"`
}

type CourseKey struct {
	CourseID uint64 `db:"course_id" csv:"-"`
}

type CourseEntity struct {
	PrimaryKey
	Course
}

type IsqEntity struct {
	PrimaryKey
	CourseKey
	Isq
}

type GradesEntity struct {
	PrimaryKey
	CourseKey
	Grades
}

type ScheduleEntity struct {
	PrimaryKey
	CourseKey
	Schedule
}

func (c CourseEntity) Persist(tx Transaction) error {
	return tx.Insert(&c)
}

func (i Isq) Persist(tx Transaction, courseKey CourseKey) error {
	return tx.Insert(&IsqEntity{CourseKey: courseKey, Isq: i})
}

func (g Grades) Persist(tx Transaction, courseKey CourseKey) error {
	return tx.Insert(&GradesEntity{CourseKey: courseKey, Grades: g})
}

func (s Schedule) Persist(tx Transaction, courseKey CourseKey) error {
	return tx.Insert(&ScheduleEntity{CourseKey: courseKey, Schedule: s})
}

func (d Dataset) Persist(tx Transaction) error {
	for course, features := range d {
		courseEntity := &CourseEntity{Course: course}
		if err := courseEntity.Persist(tx); err != nil {
			return err
		}
		courseKey := CourseKey{courseEntity.ID}
		for _, feature := range features {
			if err := feature.Persist(tx, courseKey); err != nil {
				return err
			}
		}
	}
	return nil
}

type SqliteStorage struct {
	dbmap *gorp.DbMap
}

func NewSqliteStorage(file string) SqliteStorage {
	storage := SqliteStorage{}

	db, err := sql.Open("sqlite3", file)
	if err != nil {
		log.Panic("Unable to connect to database: ", err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(CourseEntity{}, "courses").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	dbmap.AddTableWithName(IsqEntity{}, "isq")
	dbmap.AddTableWithName(GradesEntity{}, "grades")
	dbmap.AddTableWithName(ScheduleEntity{}, "sections")
	dbmap.CreateTablesIfNotExists()
	storage.dbmap = dbmap
	return storage
}

func (d SqliteStorage) Save(v Persistable) error {
	tx, err := d.dbmap.Begin()
	if err != nil {
		return err
	}
	err = v.Persist(InsertIgnoringDupes(tx))
	if err != nil {
		return err
	}
	return tx.Commit()
}
