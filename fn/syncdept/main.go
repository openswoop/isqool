package syncdept

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/scrape"
	"net/http"
)

func SyncDepartment(w http.ResponseWriter, _ *http.Request) {
	// Set up colly
	c := colly.NewCollector()
	c.AllowURLRevisit = true

	department, err := scrape.GetDepartment(c, "Spring 2019", 6502)
	if err != nil {
		panic(err)
	}

	// TODO: upload to BigQuery and fetch child courses
	spew.Fdump(w, department)
}
