package report

import (
	"cloud.google.com/go/bigquery"
	"fmt"
	"github.com/openswoop/isqool/pkg/scrape"
	"strconv"
)

type departmentViewFull struct {
	Status  string `csv:"status"`
	Crn     string `csv:"crn"`
	Course  string `csv:"course"`
	Title   string `csv:"title"`
	Credits string `csv:"credits"`
	DepartmentViewPartial
	WaitCount  string `csv:"wait_count"`
	Approval   string `csv:"approval"`
	Instructor string `csv:"instructor"`
}

type DepartmentViewPartial struct {
	PartOfTerm string `csv:"part_of_term"`
	BeginDate  string `csv:"begin_date"`
	EndDate    string `csv:"end_date"`
	Days       string `csv:"days"`
	BeginTime  string `csv:"begin_time"`
	EndTime    string `csv:"end_time"`
	MeetType   string `csv:"meet_type"`
	Building   string `csv:"building"`
	Room       string `csv:"room"`
	Campus     string `csv:"campus"`
}

func WriteDepartment(name string, d []scrape.DeptSchedule) error {
	var rows []departmentViewFull
	for _, course := range d {
		for i, meeting := range course.Meetings {
			isContinuationRow := i > 0

			partial := DepartmentViewPartial{
				PartOfTerm: course.PartOfTerm,
				BeginDate:  meeting.BeginDate.String(),
				EndDate:    meeting.EndDate.String(),
				Days:       parseNullString(meeting.Days),
				BeginTime:  parseNullTime(meeting.BeginTime),
				EndTime:    parseNullTime(meeting.EndTime),
				MeetType:   meeting.Type,
				Building:   parseNullString(meeting.Building),
				Room:       parseNullInt64(meeting.Room),
				Campus:     course.Campus,
			}

			if !isContinuationRow {
				rows = append(rows, departmentViewFull{
					Status:                parseNullString(course.Status),
					Crn:                   strconv.Itoa(course.Crn),
					Course:                course.Name,
					Title:                 course.Title,
					Credits:               strconv.Itoa(course.Credits),
					DepartmentViewPartial: partial,
					WaitCount:             strconv.Itoa(course.WaitCount),
					Approval:              parseNullString(course.Approval),
					Instructor:            parseNullString(course.Instructor),
				})
			} else {
				rows = append(rows, departmentViewFull{
					DepartmentViewPartial: partial,
				})
			}
		}
	}

	return WriteCsv(rows, name+".csv")
}

func parseNullString(n bigquery.NullString) string {
	if !n.Valid {
		return ""
	}
	return fmt.Sprint(n.StringVal)
}

func parseNullTime(n bigquery.NullTime) string {
	if !n.Valid {
		return ""
	}
	return bigquery.CivilTimeString(n.Time)
}

func parseNullInt64(n bigquery.NullInt64) string {
	if !n.Valid {
		return ""
	}
	return fmt.Sprint(n.Int64)
}
