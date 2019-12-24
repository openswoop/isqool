package database

import (
	"github.com/rothso/isqool/pkg/scrape"
	"io"
)

type Database interface {
	io.Closer
	SaveIsqs([]scrape.Isq) error
	SaveGrades([]scrape.Grades) error
	SaveSchedules([]scrape.Schedule) error
}
