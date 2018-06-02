package main

import (
	"github.com/PuerkitoBio/goquery"
	"errors"
	"strings"
	"reflect"
	"fmt"
	"strconv"
)

type Resolver func(Dataset, string) error

type ResolverMap map[reflect.Type]Resolver

func (r ResolverMap) Register(f Feature, resolver Resolver) {
	r[reflect.TypeOf(f)] = resolver
}

func (r ResolverMap) Resolve(course string, features []Feature) (Dataset, error) {
	data := make(Dataset)
	for _, f := range features {
		t := reflect.TypeOf(f)
		if resolveFunc, ok := r[t]; ok {
			err := resolveFunc(data, course)
			if err != nil {
				return nil, err
			}
		} else {
			s := fmt.Sprintf("No Resolvers registered for %s", t)
			return nil, errors.New(s)
		}
	}
	return data, nil
}

func ResolveIsq(ds Dataset, course string) error {
	urlBase := "https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id="
	doc, err := goquery.NewDocument(urlBase + course)
	if err != nil {
		return err
	}
	return UnmarshalIsq(ds, doc)
}

func UnmarshalIsq(ds Dataset, doc *goquery.Document) error {
	// Select all rows except the two header rows
	rows := doc.Find("table.datadisplaytable:nth-child(9) tr:nth-child(n+3)")
	courseCode := doc.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()

	// Fail on empty results
	size := rows.Size()
	if size == 0 {
		return errors.New("No data found for course " + courseCode)
	}

	rows.Each(func(_ int, s *goquery.Selection) {
		cells := s.Find("td")
		course := Course{
			Name:       courseCode,
			Term:       cells.Eq(0).Text(),
			Crn:        cells.Eq(1).Text(),
			Instructor: strings.TrimSpace(cells.Eq(2).Text()),
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
		ds[course] = append(ds[course], data)
	})

	return nil;
}

func ResolveGrades(ds Dataset, course string) error {
	urlBase := "https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id="
	doc, err := goquery.NewDocument(urlBase + course)
	if err != nil {
		return err
	}
	return UnmarshalGrades(ds, doc)
}

func UnmarshalGrades(ds Dataset, doc *goquery.Document) error {
	// Select all rows from the "Grade Distribution Percentages" table except the headers
	rows := doc.Find("table.datadisplaytable:nth-child(14) tr:nth-child(n+3)")
	courseCode := doc.Find("table.datadisplaytable:nth-child(5) .dddefault").First().Text()

	// Fail on empty results
	size := rows.Size()
	if size == 0 {
		return errors.New("No grades found for course " + courseCode)
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

		course := Course{
			Name:       courseCode,
			Term:       cells.Eq(0).Text(),
			Crn:        cells.Eq(1).Text(),
			Instructor: strings.TrimSpace(cells.Eq(2).Text()),
		}
		grades := Grades{
			PercentA: percentA + percentAMinus,
			PercentB: percentB + percentBMinus + percentBPlus,
			PercentC: percentC + percentCPlus,
			PercentD: percentD,
			PercentF: percentF,
			Average:  strings.TrimSpace(cells.Eq(14).Text()),
		}
		ds[course] = append(ds[course], grades)
	})

	return nil;
}

func ResolveSchedule(ds Dataset, course string) error {
	// URL to scrape
	subject := course[0:3]
	courseId := course[3:]
	urlBase := "https://banner.unf.edu/pls/nfpo/bwckctlg.p_disp_listcrse?schd_in=" +
		"&subj_in=" + subject + "&crse_in=" + courseId + "&term_in="

	// Terms to scrape
	var termIds []int
	for _, term := range termIds {
		doc, err := goquery.NewDocument(urlBase + strconv.Itoa(term))
		if err != nil {
			return err
		}
		if err := UnmarshalSchedule(ds, doc); err != nil {
			return err
		}
	}

	return nil;
}

func UnmarshalSchedule(ds Dataset, doc *goquery.Document) (error) {
	tables := doc.Find("table.datadisplaytable:nth-child(5) > tbody > tr:nth-child(even)")

	tables.Each(func(_ int, s *goquery.Selection) {
		rows := s.Find("td table.datadisplaytable tr").FilterFunction(func(_ int, s *goquery.Selection) bool {
			// Ignore any laboratory information, for now
			return strings.Contains(s.Find("td").First().Text(), "Class")
		})


	})
}