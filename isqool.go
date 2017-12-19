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
	"time"
	"regexp"
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

type ScheduleDetail struct {
	TermId     string  `csv:"-"`
	Crn        string  `csv:"-"`
	Instructor string  `csv:"-"`
	StartTime  string  `csv:"start_time"`
	Duration   string `csv:"duration"`
	Days       string  `csv:"days"`
	Building   string  `csv:"building"`
	Room       string  `csv:"room"`
	Credits    string  `csv:"credits"`
}

type Record struct {
	IsqData
	GradeDistribution
}

type ScheduledRecord struct {
	Record
	ScheduleDetail
}

func (isq IsqData) courseKey() string {
	term, _ := termToId(isq.Term)
	return strconv.Itoa(term) + " " + isq.Crn + " " + isq.Instructor
}

func (dist GradeDistribution) courseKey() string {
	term, _ := termToId(dist.Term)
	return strconv.Itoa(term) + " " + dist.Crn + " " + dist.Instructor
}

func (schd ScheduleDetail) courseKey() string {
	return schd.TermId + " " + schd.Crn + " " + schd.Instructor
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

	// Collect all the term IDs we need to query
	termsFound := make(map[string]bool)
	termIds := make([]int, 0)
	for _, record := range records {
		term := record.IsqData.Term
		if _, ok := termsFound[term]; !ok {
			termId, _ := termToId(term)
			termIds = append(termIds, termId)
			termsFound[term] = true
		}
	}

	// Get this course's scheduling data for the selected terms
	scheduleDetails, _ := getScheduleDetails(course, termIds)

	// Merge the schedule details with the records
	scheduleDetailsMap := make(map[string]ScheduleDetail)
	for _, detail := range scheduleDetails {
		scheduleDetailsMap[detail.courseKey()] = *detail
	}

	// Merge the schedule details with the records
	newRecords := make([]ScheduledRecord, len(records))
	for i, record := range records {
		withDetail := scheduleDetailsMap[record.IsqData.courseKey()]
		newRecords[i] = ScheduledRecord{*record, withDetail}
	}

	// Output to file
	fileName := course + ".csv"
	fmt.Println("Saving to", fileName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	err = gocsv.MarshalFile(newRecords, file)
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

	urlBase := "https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id="
	col.Visit(urlBase + course)

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

func getScheduleDetails(course string, termIds []int) ([]*ScheduleDetail, error) {
	col := colly.NewCollector()
	col.CacheDir = "./.section-cache"

	subject := course[0:3]
	courseId := course[3:]
	urlBase := "https://banner.unf.edu/pls/nfpo/bwckctlg.p_disp_listcrse?schd_in=" +
		"&subj_in=" + subject + "&crse_in=" + courseId + "&term_in="

	var details []*ScheduleDetail

	selector := "table.datadisplaytable:nth-child(5) > tbody > tr:nth-child(even)"
	col.OnHTML(selector, func(e *colly.HTMLElement) {
		rows := e.DOM.Find("td table.datadisplaytable tr")
		data := rows.FilterFunction(func(i int, s *goquery.Selection) bool {
			// Ignore any laboratory information, for now
			return strings.Contains(s.Find("td").First().Text(), "Class")
		}).Find("td")

		// Extract the instructor's last name
		instructorR := regexp.MustCompile(`\s((?:de )?[\w-]+) \(P\)`)
		fmt.Println(e.Request.Ctx.Get("term"), data.Last().Text())
		instructor := instructorR.FindStringSubmatch(data.Last().Text())[1]

		// Extract the start time and class duration
		var startTime, duration string
		timeText := data.Eq(1).Text()
		if timeText != "TBA" {
			times := strings.Split(data.Eq(1).Text(), " - ")
			timeBegin, _ := time.Parse("3:04 pm", times[0])
			timeEnd, _ := time.Parse("3:04 pm", times[1])
			difference := timeEnd.Sub(timeBegin).Minutes()
			startTime = timeBegin.Format("1504")
			duration = strconv.FormatFloat(difference, 'f', -1, 64)
		}

		// Extract the days the class meets
		days := strings.TrimSpace(data.Eq(2).Text())

		// Extract the building number and room number
		var building, room string
		locationText := strings.TrimSpace(data.Eq(3).Text())
		if locationText != "Online" && locationText != "Off Main Campus" {
			locationR := regexp.MustCompile(`(\d+)-[a-zA-Z\s.&-]+(\d+)`)
			location := locationR.FindStringSubmatch(locationText)
			building = location[1]
			room = location[2]
		} else {
			building = locationText
		}

		// Extract the number of credits the course is worth
		creditsR := regexp.MustCompile(`([\d])\.000 Credits`)
		credits := creditsR.FindStringSubmatch(e.DOM.Text())[1]

		detail := ScheduleDetail{
			TermId:     e.Request.Ctx.Get("term"),
			Crn:        strings.Split(e.DOM.Prev().Text(), " - ")[1],
			Instructor: instructor,
			StartTime:  startTime,
			Duration:   duration,
			Days:       days,
			Building:   building,
			Room:       room,
			Credits:    credits,
		}
		details = append(details, &detail)
	})

	for _, termId := range termIds {
		url := urlBase + strconv.Itoa(termId)
		ctx := colly.NewContext()
		ctx.Put("term", strconv.Itoa(termId))
		col.Request("GET", url, nil, ctx, nil)
	}

	return details, nil
}

// termToId takes a term string like "Fall 2017" and determines its
// corresponding id (e.g: 201780)
func termToId(term string) (int, error) {
	split := strings.Split(term, " ")

	season := split[0]
	year, err := strconv.Atoi(split[1])
	if err != nil {
		return 0, errors.New(term + " is not a valid term")
	}

	var seasonSuffix int
	switch season {
	case "Spring":
		seasonSuffix = 1
	case "Summer":
		seasonSuffix = 5
	case "Fall":
		seasonSuffix = 8
	default:
		return 0, errors.New(term + " is not a valid term")
	}

	// After Spring 2014, the season digit is in the 10s place
	if year >= 2014 && term != "Spring 2014" {
		seasonSuffix *= 10
	}

	id := year*100 + seasonSuffix
	return id, nil
}
