package report

import (
	"github.com/gocarina/gocsv"
	"github.com/rothso/isqool/pkg/scrape"
	"os"
)

type CsvCourse struct {
	Name       string `csv:"course"`
	Term       string `csv:"term"`
	Crn        int    `csv:"crn"`
	Instructor string `csv:"instructor"`
}

func toCsvCourse(c scrape.Course) CsvCourse {
	instructor := ""
	if c.Instructor.Valid {
		instructor = c.Instructor.StringVal
	}
	return CsvCourse{c.Name, c.Term, c.Crn, instructor}
}

func WriteCsv(in interface{}, fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	err = gocsv.Marshal(in, file)
	if err != nil {
		panic(err)
	}
	return file.Close()
}
