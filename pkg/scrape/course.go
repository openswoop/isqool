package scrape

type Course struct {
	Name       string `db:"name" csv:"course"`
	Term       string `db:"term" csv:"term"`
	Crn        string `db:"crn" csv:"crn"`
	Instructor string `db:"instructor" csv:"instructor"`
}
