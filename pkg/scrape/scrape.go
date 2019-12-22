package scrape

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

const BannerUrl = "https://banner.unf.edu/pls/nfpo/"

type Unmarshaler interface {
	UnmarshalDoc(doc *goquery.Document) error
}

type Scrapable interface {
	Urls() []string
	Unmarshaler
}

func Scrape(c *colly.Collector, s Scrapable) error {
	var e error
	c = c.Clone() // same collector but without old callbacks
	c.OnResponse(func(res *colly.Response) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body))
		if err != nil {
			e = err
			return
		}
		e = s.UnmarshalDoc(doc)
	})

	urls := s.Urls()
	for _, url := range urls {
		_ = c.Visit(url)
		if e != nil {
			return e
		}
	}
	return e
}
