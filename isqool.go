package main

import (
	"os"
	"github.com/gocolly/colly"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"strconv"
	"github.com/gocarina/gocsv"
	"errors"
	"time"
	"regexp"
	"log"
)

type CourseId struct {
	Term       string `csv:"term"`
	Crn        string `csv:"crn"`
	Instructor string `csv:"instructor"`
}

type IsqData struct {
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
	PercentA float32 `csv:"A"`
	PercentB float32 `csv:"B"`
	PercentC float32 `csv:"C"`
	PercentD float32 `csv:"D"`
	PercentF float32 `csv:"F"`
	Average  string  `csv:"average_gpa"`
}

type ScheduleDetail struct {
	StartTime string `csv:"start_time"`
	Duration  string `csv:"duration"`
	Days      string `csv:"days"`
	Building  string `csv:"building"`
	Room      string `csv:"room"`
	Credits   string `csv:"credits"`
}

type Record struct {
	CourseId
	Course string `csv:"course"`
	IsqData
	GradeDistribution
	ScheduleDetail
}

var cacheDir = "./.webcache"

func main() {
	course := os.Args[1] // COT3100, etc.

	// Get all past ISQ scores and grade distributions for the course
	isqData, distData, err := getCourseRecords(course)
	if err != nil {
		log.Println(err)
		os.Exit(0)
	}

	// Collect all the unique terms/semesters we need to query
	termsFound := make(map[string]bool)
	terms := make([]string, 0)
	for id := range isqData {
		term := id.Term
		if _, found := termsFound[term]; !found {
			terms = append(terms, term)
			termsFound[term] = true
		}
	}

	// Get this course's scheduling data for the selected terms
	scheduleData, _ := getScheduleDetails(course, terms)

	// Merge the ISQ records, grade distributions, and schedule details by CourseId,
	// only keeping records that have all three parts (i.e. the union set)
	records := make([]Record, 0)
	for id, isq := range isqData {
		if dist, ok := distData[id]; ok {
			if schedule, ok := scheduleData[id]; ok {
				records = append(records, Record{id, course, isq, dist, schedule})
			} else {
				// If this happens, the schedule parser is b0rked
				class := id.Term + " " + id.Crn + " " + id.Instructor
				panic("Unable to match schedule data for " + class)
			}
		} else {
			// TODO handle labs? (they don't have grade data)
			log.Println("Omitting", id, "(no grades)")
		}
	}

	// Output to file
	log.Println("Found", len(records), "records")
	log.Println("Saving to", course+".csv")
	if err := saveToCsv(course, records); err != nil {
		panic(err)
	}
}

func saveToCsv(name string, data interface{}) error {
	file, err := os.Create(name + ".csv")
	defer file.Close()
	if err != nil {
		return err
	}
	return gocsv.MarshalFile(data, file)
}

func getCourseRecords(course string) (map[CourseId]IsqData, map[CourseId]GradeDistribution, error) {
	col := colly.NewCollector()
	col.CacheDir = cacheDir

	// Map by courseKey => data so we can later join the data sets
	isqData := make(map[CourseId]IsqData)
	distData := make(map[CourseId]GradeDistribution)

	// Download the "Instructional Satisfaction Questionnaire" table
	col.OnHTML("table.datadisplaytable:nth-child(9)", func(e *colly.HTMLElement) {
		// Read each row sequentially, skipping the two header rows
		e.DOM.Find("tr:nth-child(n+3)").Each(func(i int, s *goquery.Selection) {
			cells := s.Find("td")
			id := CourseId{
				Term:       cells.Eq(0).Text(),
				Crn:        cells.Eq(1).Text(),
				Instructor: strings.TrimSpace(cells.Eq(2).Text()),
			}
			data := IsqData{
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
			isqData[id] = data
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

			id := CourseId{
				Term:       cells.Eq(0).Text(),
				Crn:        cells.Eq(1).Text(),
				Instructor: strings.TrimSpace(cells.Eq(2).Text()),
			}
			dist := GradeDistribution{
				PercentA: percentA + percentAMinus,
				PercentB: percentB + percentBMinus + percentBPlus,
				PercentC: percentC + percentCPlus,
				PercentD: percentD,
				PercentF: percentF,
				Average:  strings.TrimSpace(cells.Eq(14).Text()),
			}
			distData[id] = dist
		})
	})

	urlBase := "https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id="
	col.Visit(urlBase + course)

	// Fail on a bad course id
	if len(isqData) == 0 {
		return nil, nil, errors.New("No data found for course " + course)
	}

	return isqData, distData, nil
}

func getScheduleDetails(course string, terms []string) (map[CourseId]ScheduleDetail, error) {
	col := colly.NewCollector()
	col.CacheDir = cacheDir

	subject := course[0:3]
	courseId := course[3:]
	urlBase := "https://banner.unf.edu/pls/nfpo/bwckctlg.p_disp_listcrse?schd_in=" +
		"&subj_in=" + subject + "&crse_in=" + courseId + "&term_in="

	schedules := make(map[CourseId]ScheduleDetail)

	selector := "table.datadisplaytable:nth-child(5) > tbody > tr:nth-child(even)"
	col.OnHTML(selector, func(e *colly.HTMLElement) {
		rows := e.DOM.Find("td table.datadisplaytable tr")
		matches := rows.FilterFunction(func(i int, s *goquery.Selection) bool {
			// Ignore any laboratory information, for now
			return strings.Contains(s.Find("td").First().Text(), "Class")
		})
		data := matches.First().Find("td")

		// Extract the instructor's last name
		instructorR := regexp.MustCompile(`\s((?:de |Von )?[\w-]+) \(P\)`)
		instructor := instructorR.FindStringSubmatch(data.Last().Text())[1]

		// Unique key for the map
		id := CourseId{
			Term:       e.Request.Ctx.Get("term"),
			Crn:        strings.Split(e.DOM.Prev().Text(), " - ")[1],
			Instructor: instructor,
		}

		// Some classes have an extra meeting on Friday at a different time than the
		// other meetings, which cannot be represented in the current CSV structure.
		if matches.Size() > 1 {
			log.Println("Warning:", id, "met at uneven times; omitting additional blocks")
		}

		// Extract the start time and class duration
		var startTime, duration string
		timeText := data.Eq(1).Text()
		if timeText != "TBA" {
			times := strings.Split(timeText, " - ")
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
			locationR := regexp.MustCompile(`([\d]+[A-Z]?)-[a-zA-Z\s.&-]+(\d+)`)
			location := locationR.FindStringSubmatch(locationText)
			building = location[1]
			room = location[2]
		} else {
			building = locationText
		}

		// Extract the number of credits the course is worth
		creditsR := regexp.MustCompile(`([\d])\.000 Credits`)
		credits := creditsR.FindStringSubmatch(e.DOM.Text())[1]

		schedule := ScheduleDetail{
			StartTime: startTime,
			Duration:  duration,
			Days:      days,
			Building:  building,
			Room:      room,
			Credits:   credits,
		}
		schedules[id] = schedule
	})

	// Request the scheduling details of this course for each term
	for _, term := range terms {
		termId, _ := termToId(term)
		ctx := colly.NewContext()
		ctx.Put("term", term)
		url := urlBase + strconv.Itoa(termId)
		col.Request("GET", url, nil, ctx, nil)
	}

	return schedules, nil
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
