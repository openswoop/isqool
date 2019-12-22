package app

import (
	"github.com/gocarina/gocsv"
	"github.com/rothso/isqool/pkg/scrape"
	"os"
)

type TotalView struct {
	scrape.Course
	scrape.Isq
	scrape.Grades
	scrape.Schedule
}

type CsvRows []TotalView

func (c *CsvRows) UnmarshalDataset(dataset scrape.Dataset) {
	for course, features := range dataset {
		view := TotalView{Course: course}
		for _, feature := range features {
			switch feature.(type) {
			case scrape.Isq:
				view.Isq = feature.(scrape.Isq)
			case scrape.Grades:
				view.Grades = feature.(scrape.Grades)
			case scrape.Schedule:
				view.Schedule = feature.(scrape.Schedule)
			}
		}
		*c = append(*c, view)
	}
}

func (c CsvRows) Len() int {
	return len(c)
}

func (c CsvRows) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c CsvRows) Less(i, j int) bool {
	aTerm, _ := scrape.TermToId(c[i].Course.Term)
	bTerm, _ := scrape.TermToId(c[j].Course.Term)
	return aTerm < bTerm
}

func SaveAsCsv(in interface{}, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	err = gocsv.Marshal(in, file)
	if err != nil {
		panic(err)
	}
	_ = file.Close()
}
