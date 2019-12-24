package main

import (
	"github.com/docopt/docopt-go"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/app"
	"github.com/rothso/isqool/pkg/scrape"
	"log"
	"os"
	"regexp"
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
	isqs, grades, err := scrape.GetIsqAndGrades(c, name, isProfessor)
	if err != nil {
		panic(err)
	}
	params := scrape.CollectScheduleParams(isqs, grades)
	schedules, err := scrape.GetSchedules(c, params)
	if err != nil {
		panic(err)
	}
	log.Println("Found", len(schedules), "records")

	// Save all the data to the database
	storage := app.NewSqliteStorage(dbFile)
	var insertionData []interface{}
	for i := range isqs {
		insertionData = append(insertionData, &isqs[i])
	}
	for i := range grades {
		insertionData = append(insertionData, &grades[i])
	}
	for i := range schedules {
		insertionData = append(insertionData, &schedules[i])
	}
	if err := storage.Save(insertionData); err != nil {
		panic(err)
	}
	_ = storage.Close()
	log.Println("Saved to database", dbFile)

	// TODO: refactor the rest of the code
	//// Also output to a csv
	//view := app.CsvRows{}
	//view.UnmarshalDataset(data)
	//sort.Sort(sort.Reverse(view))
	//app.SaveAsCsv(view, name+".csv")
	//log.Println("Wrote to file", name+".csv")
}
