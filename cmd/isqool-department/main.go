package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/scrape"
)

func main() {
	// Set up colly
	c := colly.NewCollector()
	c.AllowURLRevisit = true

	dept, err := scrape.GetDepartment(c, "Spring 2019", 6502)
	if err != nil {
		panic(err)
	}

	spew.Dump(dept)
}
