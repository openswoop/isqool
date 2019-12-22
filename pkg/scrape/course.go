package scrape

import "github.com/rothso/isqool/pkg/persist"

type Course struct {
	Name       string `db:"name" csv:"course"`
	Term       string `db:"term" csv:"term"`
	Crn        string `db:"crn" csv:"crn"`
	Instructor string `db:"instructor" csv:"instructor"`
}

type CourseEntity struct {
	PrimaryKey
	Course
}

func (c CourseEntity) Persist(tx persist.Transaction) error {
	return tx.Insert(&c)
}
