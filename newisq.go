package main

import (
	"os"
	"github.com/gocolly/colly"
	"log"
)

type Dataset map[Course][]Feature

type MapperFunc func(Dataset) (Dataset, error)

func (d *Dataset) Apply(mapperFunc MapperFunc) {
	res, err := mapperFunc(*d)
	if err != nil {
		panic(err)
	}
	*d = res
}

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

	// Save to the database
	storage, err := NewSqliteStorage(dbFile)
	if err != nil {
		panic(err)
	}
	err = storage.Save(data)
	if err != nil {
		panic(err)
	}
}
