package scrape

import (
	"fmt"
	"github.com/rothso/isqool/pkg/persist"
	"os"
)

type Dataset map[Course][]Feature

type Feature interface {
	Persist(tx persist.Transaction, courseKey CourseKey) error
}

type PrimaryKey struct {
	ID uint64 `db:"id, primarykey, autoincrement" csv:"-"`
}

type CourseKey struct {
	CourseID uint64 `db:"course_id" csv:"-"`
}

func (d Dataset) Persist(tx persist.Transaction) error {
	for course, features := range d {
		courseEntity := &CourseEntity{Course: course}
		if err := courseEntity.Persist(tx); err != nil {
			return err
		}
		courseKey := CourseKey{courseEntity.ID}
		for _, feature := range features {
			if err := feature.Persist(tx, courseKey); err != nil {
				return err
			}
		}
	}
	return nil
}

type MapFunc func(Dataset) (Dataset, error)

func (d *Dataset) Apply(mapFunc MapFunc) {
	res, err := mapFunc(*d)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	*d = res
}

func RemoveLabs() MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		for course, features := range dataset {
			var hasIsq, hasGrades bool
			for _, feature := range features {
				switch feature.(type) {
				case Isq:
					hasIsq = true
				case Grades:
					hasGrades = true
				}
			}

			// Labs have professor ISQ scores but no grade distributions
			if hasIsq && !hasGrades {
				delete(dataset, course)
			}
		}
		return dataset, nil
	}
}
