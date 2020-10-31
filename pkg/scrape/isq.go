package scrape

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"strings"
)

type Isq struct {
	Enrolled     int     `bigquery:"enrolled" db:"enrolled" csv:"enrolled"`
	Responded    int     `bigquery:"responded" db:"responded" csv:"responded"`
	ResponseRate float64 `bigquery:"response_rate" db:"response_rate" csv:"response_rate"`
	Percent5     float64 `bigquery:"percent_5" db:"percent_5" csv:"percent_5"`
	Percent4     float64 `bigquery:"percent_4" db:"percent_4" csv:"percent_4"`
	Percent3     float64 `bigquery:"percent_3" db:"percent_3" csv:"percent_3"`
	Percent2     float64 `bigquery:"percent_2" db:"percent_2" csv:"percent_2"`
	Percent1     float64 `bigquery:"percent_1" db:"percent_1" csv:"percent_1"`
	Rating       float64 `bigquery:"rating" db:"rating" csv:"rating"`
}

type Grades struct {
	PercentA float64 `bigquery:"percent_a" db:"percent_a" csv:"A"`
	PercentB float64 `bigquery:"percent_b" db:"percent_b" csv:"B"`
	PercentC float64 `bigquery:"percent_c" db:"percent_c" csv:"C"`
	PercentD float64 `bigquery:"percent_d" db:"percent_d" csv:"D"`
	PercentF float64 `bigquery:"percent_e" db:"percent_e" csv:"F"`
	Average  float64 `bigquery:"average_gpa" db:"average_gpa" csv:"average_gpa"`
}

type CourseIsq struct {
	Course
	Isq
}

type CourseGrades struct {
	Course
	Grades
}

func GetIsqAndGrades(c *colly.Collector, name string, isProfessor bool) ([]CourseIsq, []CourseGrades, error) {
	var isqs []CourseIsq
	var grades []CourseGrades

	// Collect the ISQ table
	c.OnHTML(".pagebodydiv", func(e *colly.HTMLElement) {
		// Select all rows except the two header rows
		rows := e.DOM.Find("table.datadisplaytable:nth-child(9) tr:nth-child(n+3)")
		headerText := e.DOM.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()

		rows.Each(func(_ int, s *goquery.Selection) {
			cells := s.Find("td")

			// Professor pages have "Course ID" in place of "Instructor"
			var courseID, instructor string
			if isProfessor {
				courseID = strings.TrimSpace(cells.Eq(2).Text())
				instructor = getLastName(headerText)
			} else {
				courseID = headerText
				instructor = strings.TrimSpace(cells.Eq(2).Text())
			}

			course := Course{
				Name:       courseID,
				Term:       cells.Eq(0).Text(),
				Crn:        atoi(cells.Eq(1).Text()),
				Instructor: nullString(instructor),
			}
			isq := Isq{
				Enrolled:     atoi(strings.TrimSpace(cells.Eq(3).Text())),
				Responded:    atoi(strings.TrimSpace(cells.Eq(4).Text())),
				ResponseRate: parseFloat(cells.Eq(5).Text()),
				Percent5:     parseFloat(cells.Eq(6).Text()),
				Percent4:     parseFloat(cells.Eq(7).Text()),
				Percent3:     parseFloat(cells.Eq(8).Text()),
				Percent2:     parseFloat(cells.Eq(9).Text()),
				Percent1:     parseFloat(cells.Eq(10).Text()),
				Rating:       parseFloat(cells.Eq(12).Text()),
			}
			isqs = append(isqs, CourseIsq{course, isq})
		})
	})

	// Collect the Grades table
	c.OnHTML(".pagebodydiv", func(e *colly.HTMLElement) {
		// Select all rows from the "Grade Distribution Percentages" table except the headers
		rows := e.DOM.Find("table.datadisplaytable:nth-child(14) tr:nth-child(n+3)")
		headerText := e.DOM.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()

		rows.Each(func(_ int, s *goquery.Selection) {
			cells := s.Find("td")
			percentA := parseFloat(cells.Eq(4).Text())
			percentAMinus := parseFloat(cells.Eq(5).Text())
			percentBPlus := parseFloat(cells.Eq(6).Text())
			percentB := parseFloat(cells.Eq(7).Text())
			percentBMinus := parseFloat(cells.Eq(8).Text())
			percentCPlus := parseFloat(cells.Eq(9).Text())
			percentC := parseFloat(cells.Eq(10).Text())
			percentD := parseFloat(cells.Eq(11).Text())
			percentF := parseFloat(cells.Eq(12).Text())
			average := parseFloat(cells.Eq(14).Text())

			// Professor pages have "Course ID" in place of "Instructor"
			var courseID, instructor string
			if isProfessor {
				courseID = strings.TrimSpace(cells.Eq(2).Text())
				instructor = getLastName(headerText)
			} else {
				courseID = headerText
				instructor = strings.TrimSpace(cells.Eq(2).Text())
			}

			course := Course{
				Name:       courseID,
				Term:       cells.Eq(0).Text(),
				Crn:        atoi(cells.Eq(1).Text()),
				Instructor: nullString(instructor),
			}
			data := Grades{
				PercentA: round(percentA + percentAMinus),
				PercentB: round(percentB + percentBMinus + percentBPlus),
				PercentC: round(percentC + percentCPlus),
				PercentD: percentD,
				PercentF: percentF,
				Average:  average,
			}
			grades = append(grades, CourseGrades{course, data})
		})
	})

	var url string
	if isProfessor {
		url = bannerUrl + "wksfwbs.p_instructor_isq_grade?pv_instructor=" + name
	} else {
		url = bannerUrl + "wksfwbs.p_course_isq_grade?pv_course_id=" + name
	}

	return isqs, grades, c.Visit(url)
}
