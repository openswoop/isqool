package main

import (
	"os"
	"github.com/gocolly/colly"
	"log"
	"sort"
	"strings"
)

type Dataset map[Course][]Feature

func (d *Dataset) Apply(mapFunc MapFunc) {
	res, err := mapFunc(*d)
	if err != nil {
		panic(err)
	}
	*d = res
}

var (
	cacheDir = "./.webcache"
	dbFile   = "isqool.db"
)

func main() {
	name := os.Args[1] // COT3100 or N00474503 etc.
	isProfessor := strings.HasPrefix(name, "N")

	// Set up colly
	c := colly.NewCollector()
	c.CacheDir = cacheDir
	c.AllowURLRevisit = true

	// Scrape the data
	data := Dataset{}
	if isProfessor {
		data.Apply(ResolveIsqByProfessor(c, name))
		data.Apply(ResolveGradesByProfessor(c, name))
	} else {
		data.Apply(ResolveIsq(c, name))
		data.Apply(ResolveGrades(c, name))
		data.Apply(ResolveSchedule(c, name))
	}
	log.Println("Found", len(data), "records")

	// Save all the data to the database
	storage := NewSqliteStorage(dbFile)
	if err := storage.Save(data); err != nil {
		panic(err)
	}
	storage.Close()
	log.Println("Saved to database", dbFile)

	// Also output to a csv
	view := CsvRows{}
	view.UnmarshalDataset(data)
	sort.Sort(sort.Reverse(view))
	SaveAsCsv(view, name+".csv")
	log.Println("Wrote to file", name+".csv")
}
