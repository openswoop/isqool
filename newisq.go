package main

import (
	"os"
	"github.com/gocolly/colly"
	"log"
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
	course := os.Args[1]

	// Set up colly
	c := colly.NewCollector()
	c.CacheDir = cacheDir

	// Scrape the data
	data := Dataset{}
	data.Apply(ResolveIsq(c, course))
	data.Apply(ResolveGrades(c, course))
	data.Apply(ResolveSchedule(c, course))
	log.Println("Found", len(data), "records")

	// Save all the data to the database
	storage := NewSqliteStorage(dbFile)
	if err := storage.Save(data); err != nil {
		panic(err)
	}
	storage.Close()
}
