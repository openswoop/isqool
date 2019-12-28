package scrape

import "cloud.google.com/go/bigquery"

type Course struct {
	Name       string              `db:"name" csv:"course"`
	Term       string              `db:"term" csv:"term"`
	Crn        int                 `db:"crn" csv:"crn"`
	Instructor bigquery.NullString `db:"instructor" csv:"instructor"`
}
