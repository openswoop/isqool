package report

import (
	"github.com/openswoop/isqool/pkg/scrape"
	"sort"
)

type courseView struct {
	CsvCourse
	scrape.Isq
	scrape.Grades
	scrape.Schedule
}

type CourseInput struct {
	Isqs      []scrape.CourseIsq
	Grades    []scrape.CourseGrades
	Schedules []scrape.CourseSchedule
}

func WriteCourse(name string, r CourseInput) error {
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
	var rows courseReport
	for course, isq := range courseToIsq {
		rows = append(rows, courseView{
			CsvCourse: toCsvCourse(course),
			Isq:       isq,
			Grades:    courseToGrades[course],
			Schedule:  courseToSchedules[course],
		})
	}

	sort.Sort(sort.Reverse(rows))
	return WriteCsv(rows, name+".csv")
}

type courseReport []courseView

func (r courseReport) Len() int {
	return len(r)
}

func (r courseReport) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r courseReport) Less(i, j int) bool {
	aTerm, _ := scrape.TermToId(r[i].CsvCourse.Term)
	bTerm, _ := scrape.TermToId(r[j].CsvCourse.Term)
	return aTerm < bTerm
}