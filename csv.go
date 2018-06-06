package main

import (
	"os"
	"github.com/gocarina/gocsv"
)

type TotalView struct {
	Course
	Isq
	Grades
	Schedule
}

type CsvRows []TotalView

func (c *CsvRows) UnmarshalDataset(dataset Dataset) {
	for course, features := range dataset {
		view := TotalView{Course: course}
		for _, feature := range features {
			switch feature.(type) {
			case Isq:
				view.Isq = feature.(Isq)
			case Grades:
				view.Grades = feature.(Grades)
			case Schedule:
				view.Schedule = feature.(Schedule)
			}
		}
		*c = append(*c, view)
	}
}

func SaveAsCsv(in interface{}, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	err = gocsv.Marshal(in, file)
	if err != nil {
		panic(err)
	}
	file.Close()
}