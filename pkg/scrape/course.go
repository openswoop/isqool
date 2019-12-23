package scrape

import "github.com/rothso/isqool/pkg/persist"

type Course struct {
	Name       string `db:"name" csv:"course"`
	Term       string `db:"term" csv:"term"`
	Crn        string `db:"crn" csv:"crn"`
	Instructor string `db:"instructor" csv:"instructor"`
}

func (course Course) Persist(tx persist.Transaction) error {
	return tx.Insert(course)
}
