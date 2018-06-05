package main

import (
	"os"
	"github.com/gocolly/colly"
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

	count := 0
	for range data {
		count++
	}

	print(count)

	//spew.Dump(data)

	// Ensure these models are in the DB
	// Open up a transaction
	//var db Database
	//db.PersistEntity(data)
	//data.Persist(NoopDb{})
}
