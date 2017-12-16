package main

import (
	"os"
	"github.com/gocolly/colly"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"fmt"
	"strconv"
	"github.com/gocarina/gocsv"
	"errors"
)

type IsqData struct {
	Term         string `csv:"term"`
	Crn          string `csv:"crn"`
	Course       string `csv:"course"`
	Instructor   string `csv:"instructor"`
	Enrolled     string `csv:"enrolled"`
	Responded    string `csv:"responded"`
	ResponseRate string `csv:"response_rate"`
	Percent5     string `csv:"percent_5"`
	Percent4     string `csv:"percent_4"`
	Percent3     string `csv:"percent_3"`
	Percent2     string `csv:"percent_2"`
	Percent1     string `csv:"percent_1"`
	Rating       string `csv:"rating"`
}

type GradeDistribution struct {
	Term       string  `csv:"-"`
	Crn        string  `csv:"-"`
	Instructor string  `csv:"-"`
	PercentA   float32 `csv:"A"`
	PercentB   float32 `csv:"B"`
	PercentC   float32 `csv:"C"`
	PercentD   float32 `csv:"D"`
	PercentF   float32 `csv:"F"`
	Average    string  `csv:"average_gpa"`
}

type Record struct {
	IsqData
	GradeDistribution
}

func (isq IsqData) courseKey() string {
	return isq.Term + " " + isq.Crn + " " + isq.Instructor
}

func (dist GradeDistribution) courseKey() string {
	return dist.Term + " " + dist.Crn + " " + dist.Instructor
}

func main() {
	course := os.Args[1] // COT3100, etc.

	// Get all past ISQ scores and grade distributions for the course
	records, err := getCourseRecords(course)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println("Found", len(records), "records.")

	// Output to file
	fileName := course + ".csv"
	fmt.Println("Saving to", fileName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	err = gocsv.MarshalFile(&records, file)
	if err != nil {
		panic(err)
	}
	fmt.Println("Success!")
}

func getCourseRecords(course string) ([]*Record, error) {
	col := colly.NewCollector()
	col.CacheDir = "./.cache"

	// Map by courseKey => data so we can later join both data sets
	isqData := make(map[string]IsqData)
	distData := make(map[string]GradeDistribution)
	var records []*Record

	// Download the "Instructional Satisfaction Questionnaire" table
	col.OnHTML("table.datadisplaytable:nth-child(9)", func(e *colly.HTMLElement) {
		// Read each row sequentially, skipping the two header rows
		e.DOM.Find("tr:nth-child(n+3)").Each(func(i int, s *goquery.Selection) {
			cells := s.Find("td")
			data := IsqData{
				Term:         cells.Eq(0).Text(),
				Crn:          cells.Eq(1).Text(),
				Course:       course,
				Instructor:   strings.TrimSpace(cells.Eq(2).Text()),
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
			isqData[data.courseKey()] = data
		})
	})

	// Simultaneously download the "Grade Distribution Percentages" table
	col.OnHTML("table.datadisplaytable:nth-child(14)", func(e *colly.HTMLElement) {
		// Read each row sequentially, skipping the two header rows
		e.DOM.Find("tr:nth-child(n+3)").Each(func(i int, s *goquery.Selection) {
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

			dist := GradeDistribution{
				Term:       cells.Eq(0).Text(),
				Crn:        cells.Eq(1).Text(),
				Instructor: strings.TrimSpace(cells.Eq(2).Text()),
				PercentA:   percentA + percentAMinus,
				PercentB:   percentB + percentBMinus + percentBPlus,
				PercentC:   percentC + percentCPlus,
				PercentD:   percentD,
				PercentF:   percentF,
				Average:    strings.TrimSpace(cells.Eq(14).Text()),
			}
			distData[dist.courseKey()] = dist
		})
	})

	url := "https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id=" + course
	col.Visit(url)

	// Fail on a bad course id
	if len(isqData) == 0 {
		return nil, errors.New("No data found for course " + course)
	}

	// Only keep ISQ records that also have grade data (i.e. find the union set)
	for key, isq := range isqData {
		// TODO: special handling for labs? (they don't have their own grade data)
		if dist, ok := distData[key]; ok {
			records = append(records, &Record{isq, dist})
		}
	}

	return records, nil
}
