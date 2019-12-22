package scrape

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/persist"
	"strconv"
	"strings"
)

type Isq struct {
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
	PercentA float32 `db:"percent_a" csv:"A"`
	PercentB float32 `db:"percent_b" csv:"B"`
	PercentC float32 `db:"percent_c" csv:"C"`
	PercentD float32 `db:"percent_d" csv:"D"`
	PercentF float32 `db:"percent_e" csv:"F"`
	Average  string  `db:"average_gpa" csv:"average_gpa"`
}

type ScrapeByCourse struct {
	Unmarshaler
	course string
}

type ScrapeByProfessor struct {
	Unmarshaler
	professor string
}

func (i ScrapeByCourse) Urls() []string {
	return []string{BannerUrl + "wksfwbs.p_course_isq_grade?pv_course_id=" + i.course}
}

func (i ScrapeByProfessor) Urls() []string {
	return []string{BannerUrl + "wksfwbs.p_instructor_isq_grade?pv_instructor=" + i.professor}
}

type ScrapeIsq struct {
	data Dataset
}

type ScrapeGrades struct {
	data Dataset
}

func (i ScrapeIsq) UnmarshalDoc(doc *goquery.Document) error {
	// Select all rows except the two header rows
	rows := doc.Find("table.datadisplaytable:nth-child(9) tr:nth-child(n+3)")

	// Figure out if we're scraping a professor page or a course page
	label := doc.Find("table.datadisplaytable:nth-child(5) .ddlabel").First().Text()
	headerText := doc.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()
	hasVarietyCourses := label == "Instructor: "

	// Fail on empty results
	size := rows.Size()
	if size == 0 {
		if hasVarietyCourses {
			return errors.New("No ISQs found for instructor " + headerText)
		} else {
			return errors.New("No ISQs found for course " + headerText)
		}
	}

	rows.Each(func(_ int, s *goquery.Selection) {
		cells := s.Find("td")

		// Professor pages have "Course ID" in place of "Instructor"
		var courseID, instructor string
		if hasVarietyCourses {
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
		data := Isq{
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
		i.data[course] = append(i.data[course], data)
	})

	return nil
}

func (g ScrapeGrades) UnmarshalDoc(doc *goquery.Document) error {
	// Select all rows from the "Grade Distribution Percentages" table except the headers
	rows := doc.Find("table.datadisplaytable:nth-child(14) tr:nth-child(n+3)")

	// Figure out if we're scraping a professor page or a course page
	label := doc.Find("table.datadisplaytable:nth-child(5) .ddlabel").First().Text()
	headerText := doc.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()
	hasVarietyCourses := label == "Instructor: "

	// Fail on empty results
	size := rows.Size()
	if size == 0 {
		if hasVarietyCourses {
			return errors.New("No grades found for instructor " + headerText)
		} else {
			return errors.New("No grades found for course " + headerText)
		}
	}

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
		if hasVarietyCourses {
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
		grades := Grades{
			PercentA: percentA + percentAMinus,
			PercentB: percentB + percentBMinus + percentBPlus,
			PercentC: percentC + percentCPlus,
			PercentD: percentD,
			PercentF: percentF,
			Average:  strings.TrimSpace(cells.Eq(14).Text()),
		}
		g.data[course] = append(g.data[course], grades)
	})

	return nil
}

func ResolveIsq(c *colly.Collector, course string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeByCourse{ScrapeIsq{dataset}, course})
		return dataset, err
	}
}

func ResolveGrades(c *colly.Collector, course string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeByCourse{ScrapeGrades{dataset}, course})
		return dataset, err
	}
}

func ResolveIsqByProfessor(c *colly.Collector, professor string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeByProfessor{ScrapeIsq{dataset}, professor})
		return dataset, err
	}
}

func ResolveGradesByProfessor(c *colly.Collector, professor string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeByProfessor{ScrapeGrades{dataset}, professor})
		return dataset, err
	}
}

type IsqEntity struct {
	PrimaryKey
	CourseKey
	Isq
}

type GradesEntity struct {
	PrimaryKey
	CourseKey
	Grades
}

func (i Isq) Persist(tx persist.Transaction, courseKey CourseKey) error {
	return tx.Insert(&IsqEntity{CourseKey: courseKey, Isq: i})
}

func (g Grades) Persist(tx persist.Transaction, courseKey CourseKey) error {
	return tx.Insert(&GradesEntity{CourseKey: courseKey, Grades: g})
}
