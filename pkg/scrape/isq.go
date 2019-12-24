package scrape

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"strconv"
	"strings"
)

type Isq struct {
	Course
	Enrolled     string `db:"enrolled" csv:"enrolled"`
	Responded    string `db:"responded" csv:"responded"`
	ResponseRate string `db:"response_rate" csv:"response_rate"`
	Percent5     string `db:"percent_5" csv:"percent_5"`
	Percent4     string `db:"percent_4" csv:"percent_4"`
	Percent3     string `db:"percent_3" csv:"percent_3"`
	Percent2     string `db:"percent_2" csv:"percent_2"`
	Percent1     string `db:"percent_1" csv:"percent_1"`
	Rating       string `db:"rating" csv:"rating"`
}

type Grades struct {
	Course
	PercentA float32 `db:"percent_a" csv:"A"`
	PercentB float32 `db:"percent_b" csv:"B"`
	PercentC float32 `db:"percent_c" csv:"C"`
	PercentD float32 `db:"percent_d" csv:"D"`
	PercentF float32 `db:"percent_e" csv:"F"`
	Average  string  `db:"average_gpa" csv:"average_gpa"`
}

func GetIsqAndGrades(c *colly.Collector, name string, isProfessor bool) ([]Isq, []Grades, error) {
	var isqs []Isq
	var grades []Grades

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
				Crn:        cells.Eq(1).Text(),
				Instructor: instructor,
			}
			isq := Isq{
				Course:       course,
				Enrolled:     strings.TrimSpace(cells.Eq(3).Text()),
				Responded:    strings.TrimSpace(cells.Eq(4).Text()),
				ResponseRate: strings.TrimSpace(cells.Eq(5).Text()),
				Percent5:     strings.TrimSpace(cells.Eq(6).Text()),
				Percent4:     strings.TrimSpace(cells.Eq(7).Text()),
				Percent3:     strings.TrimSpace(cells.Eq(8).Text()),
				Percent2:     strings.TrimSpace(cells.Eq(9).Text()),
				Percent1:     strings.TrimSpace(cells.Eq(10).Text()),
				Rating:       strings.TrimSpace(cells.Eq(12).Text()),
			}
			isqs = append(isqs, isq)
		})
	})

	// Collect the Grades table
	c.OnHTML(".pagebodydiv", func(e *colly.HTMLElement) {
		// Select all rows from the "Grade Distribution Percentages" table except the headers
		rows := e.DOM.Find("table.datadisplaytable:nth-child(14) tr:nth-child(n+3)")
		headerText := e.DOM.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()

		rows.Each(func(_ int, s *goquery.Selection) {
			cells := s.Find("td")
			parse := func(s string) float32 {
				float, _ := strconv.ParseFloat(strings.TrimSpace(s), 32)
				return float32(float)
			}
			percentA := parse(cells.Eq(4).Text())
			percentAMinus := parse(cells.Eq(5).Text())
			percentBPlus := parse(cells.Eq(6).Text())
			percentB := parse(cells.Eq(7).Text())
			percentBMinus := parse(cells.Eq(8).Text())
			percentCPlus := parse(cells.Eq(9).Text())
			percentC := parse(cells.Eq(10).Text())
			percentD := parse(cells.Eq(11).Text())
			percentF := parse(cells.Eq(12).Text())

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
				Crn:        cells.Eq(1).Text(),
				Instructor: instructor,
			}
			data := Grades{
				Course:   course,
				PercentA: percentA + percentAMinus,
				PercentB: percentB + percentBMinus + percentBPlus,
				PercentC: percentC + percentCPlus,
				PercentD: percentD,
				PercentF: percentF,
				Average:  strings.TrimSpace(cells.Eq(14).Text()),
			}
			grades = append(grades, data)
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
