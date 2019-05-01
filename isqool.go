package main

import (
	"github.com/docopt/docopt-go"
	"github.com/gocolly/colly"
	"log"
	"regexp"
	"sort"
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
	usage := `ISQ Scraper.

Usage:
  isqool <name>
  isqool -h | --help

Options:
  -h --help       Show this screen.
  --version       Show version.`

	opts, _ := docopt.ParseArgs(usage, nil, "1.0.0rc1")

	name, _ := opts.String("<name>") // COT3100 or N00474503 etc.
	isProfessor, _ := regexp.MatchString("N\\d{8}", name)

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
	}
	data.Apply(RemoveLabs())
	data.Apply(ResolveSchedule(c))
	log.Println("Found", len(data), "records")

	// Save all the data to the database
	storage := NewSqliteStorage(dbFile)
	if err := storage.Save(data); err != nil {
		panic(err)
	}
	_ = storage.Close()
	log.Println("Saved to database", dbFile)

	// Also output to a csv
	view := CsvRows{}
	view.UnmarshalDataset(data)
	sort.Sort(sort.Reverse(view))
	SaveAsCsv(view, name+".csv")
	log.Println("Wrote to file", name+".csv")
}
