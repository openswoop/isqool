package main

import (
	"github.com/docopt/docopt-go"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/app"
	"github.com/rothso/isqool/pkg/scrape"
	"log"
	"os"
	"regexp"
	"sort"
)

var (
	cacheDir string
	dbFile   string
)

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	cacheDir = userCacheDir + "/isqool/web-cache"
	dbFile = userCacheDir + "/isqool/isqool.db"
}

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
	data := scrape.Dataset{}
	if isProfessor {
		data.Apply(scrape.ResolveIsqByProfessor(c, name))
		data.Apply(scrape.ResolveGradesByProfessor(c, name))
	} else {
		data.Apply(scrape.ResolveIsq(c, name))
		data.Apply(scrape.ResolveGrades(c, name))
	}
	data.Apply(scrape.RemoveLabs())
	data.Apply(scrape.ResolveSchedule(c))
	log.Println("Found", len(data), "records")

	// Save all the data to the database
	storage := app.NewSqliteStorage(dbFile)
	if err := storage.Save(data); err != nil {
		panic(err)
	}
	_ = storage.Close()
	log.Println("Saved to database", dbFile)

	// Also output to a csv
	view := app.CsvRows{}
	view.UnmarshalDataset(data)
	sort.Sort(sort.Reverse(view))
	app.SaveAsCsv(view, name+".csv")
	log.Println("Wrote to file", name+".csv")
}
