package app

import (
	"github.com/gocarina/gocsv"
	"github.com/rothso/isqool/pkg/scrape"
	"os"
	"sort"
)

type reportView struct {
	scrape.Course
	scrape.Isq
	scrape.Grades
	scrape.Schedule
}

type ReportInput struct {
	Isqs      []scrape.CourseIsq
	Grades    []scrape.CourseGrades
	Schedules []scrape.CourseSchedule
}

func SaveReport(name string, r ReportInput) error {
	courseToIsq := make(map[scrape.Course]scrape.Isq, len(r.Isqs))
	for _, v := range r.Isqs {
		courseToIsq[v.Course] = v.Isq
	}
	courseToGrades := make(map[scrape.Course]scrape.Grades, len(r.Isqs))
	for _, v := range r.Grades {
		courseToGrades[v.Course] = v.Grades
	}
	courseToSchedules := make(map[scrape.Course]scrape.Schedule, len(r.Isqs))
	for _, v := range r.Schedules {
		courseToSchedules[v.Course] = v.Schedule
	}

	// Left join grades and schedules to isqs
	var rows report
	for course, isq := range courseToIsq {
		rows = append(rows, reportView{
			Course:   course,
			Isq:      isq,
			Grades:   courseToGrades[course],
			Schedule: courseToSchedules[course],
		})
	}

	sort.Sort(sort.Reverse(rows))
	return WriteCsv(rows, name+".csv")
}

type report []reportView

func (r report) Len() int {
	return len(r)
}

func (r report) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r report) Less(i, j int) bool {
	aTerm, _ := scrape.TermToId(r[i].Course.Term)
	bTerm, _ := scrape.TermToId(r[j].Course.Term)
	return aTerm < bTerm
}

func WriteCsv(in interface{}, fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	err = gocsv.Marshal(in, file)
	if err != nil {
		panic(err)
	}
	return file.Close()
}
