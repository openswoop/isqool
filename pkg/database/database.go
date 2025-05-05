package database

import (
	"github.com/openswoop/isqool/pkg/scrape"
	"io"
)

type Database interface {
	io.Closer
	SaveIsqs([]scrape.CourseIsq) error
	SaveGrades([]scrape.CourseGrades) error
	SaveSchedules([]scrape.CourseSchedule) error
}
