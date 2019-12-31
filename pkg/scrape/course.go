package scrape

type Course struct {
	Name       string     `db:"name" csv:"course" bigquery:"course"`
	Term       string     `db:"term" csv:"term" bigquery:"term"`
	Crn        int        `db:"crn" csv:"crn" bigquery:"crn"`
	Instructor NullString `db:"instructor" csv:"instructor" bigquery:"instructor"`
}
