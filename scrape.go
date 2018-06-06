package main

import (
	"github.com/PuerkitoBio/goquery"
	"errors"
	"strings"
	"strconv"
	"time"
	"regexp"
	"log"
	"github.com/gocolly/colly"
	"bytes"
)

type Unmarshaler interface {
	UnmarshalDoc(doc *goquery.Document) error
}

type Scrapable interface {
	Urls() []string
	Unmarshaler
}

func Scrape(c *colly.Collector, s Scrapable) error {
	var e error
	c = c.Clone() // same collector but without old callbacks
	c.OnResponse(func(res *colly.Response) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body))
		if err != nil {
			e = err
			return
		}
		e = s.UnmarshalDoc(doc)
	})

	urls := s.Urls()
	for _, url := range urls {
		c.Visit(url)
		if e != nil {
			return e
		}
	}
	return e
}

type MapFunc func(Dataset) (Dataset, error)

func ResolveIsq(c *colly.Collector, course string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeIsq{course, dataset})
		return dataset, err
	}
}

func ResolveGrades(c *colly.Collector, course string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeGrades{course, dataset})
		return dataset, err
	}
}

func ResolveSchedule(c *colly.Collector, course string) MapFunc {
	return func(dataset Dataset) (Dataset, error) {
		err := Scrape(c, ScrapeSchedule{course, dataset})
		return dataset, err
	}
}

type ScrapeIsq struct {
	course string
	data Dataset
}

func (i ScrapeIsq) UnmarshalDoc(doc *goquery.Document) error {
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
		i.data[course] = append(i.data[course], data)
	})

	return nil
}

func (i ScrapeIsq) Urls() []string {
	return []string{"https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id=" + i.course}
}

type ScrapeGrades struct {
	course string
	data Dataset
}

func (g ScrapeGrades) UnmarshalDoc(doc *goquery.Document) error {
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
		g.data[course] = append(g.data[course], grades)
	})

	return nil
}

func (g ScrapeGrades) Urls() []string {
	return []string{"https://banner.unf.edu/pls/nfpo/wksfwbs.p_course_isq_grade?pv_course_id=" + g.course}
}

type ScrapeSchedule struct {
	course string
	data   Dataset
}

func (sch ScrapeSchedule) Urls() []string {
	subject := sch.course[0:3]
	courseId := sch.course[3:]
	urlBase := "https://banner.unf.edu/pls/nfpo/bwckctlg.p_disp_listcrse?schd_in=" +
		"&subj_in=" + subject + "&crse_in=" + courseId + "&term_in="

	// Collect all the unique terms/semesters we need to query
	termsFound := make(map[int]bool)
	urls := make([]string, 0)
	for id := range sch.data {
		term, err := termToId(id.Term)
		if err != nil {
			continue
		}
		if _, found := termsFound[term]; !found {
			urls = append(urls, urlBase + strconv.Itoa(term))
			termsFound[term] = true
		}
	}

	return urls
}

func (sch ScrapeSchedule) UnmarshalDoc(doc *goquery.Document) error {
	tables := doc.Find("table.datadisplaytable:nth-child(5) > tbody > tr:nth-child(even)")
	term := strings.TrimSpace(doc.Find(".staticheaders").Contents().Eq(0).Text())

	tables.Each(func(_ int, s *goquery.Selection) {
		rows := s.Find("td table.datadisplaytable tr").FilterFunction(func(_ int, s *goquery.Selection) bool {
			// Ignore any laboratory information, for now
			return strings.Contains(s.Find("td").First().Text(), "Class")
		})

		// Unique key for the map
		headerData := strings.Split(s.Prev().Text(), " - ")
		course := Course{
			Name: strings.Replace(headerData[2], " ", "", 1),
			Term: term,
			Crn:  headerData[1],
		}

		if rows.Size() == 0 {
			// Some classes are "hybrid" classes that only meet on certain weeks of the
			// month, which require special parsing. Will be implemented soon.
			log.Println("Warning:", course, "has a hybrid schedule; omitting scheduling data")
			return // TODO: collapse all hybrid information into one record
		} else if rows.Size() > 1 {
			// Some classes have an extra meeting on Friday at a different time than the
			// other meetings, which cannot be represented in the current CSV structure.
			log.Println("Warning:", course, "met at uneven times; omitting additional blocks")
		}

		data := rows.First().Find("td")

		// Extract the instructor's last name
		instructorR := regexp.MustCompile(`\s((?:de |Von )?[\w-]+) \(P\)`)
		instructor := instructorR.FindStringSubmatch(data.Last().Text())[1]
		course.Instructor = instructor

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
		credits := creditsR.FindStringSubmatch(s.Text())[1]

		schedule := Schedule{
			StartTime: startTime,
			Duration:  duration,
			Days:      days,
			Building:  building,
			Room:      room,
			Credits:   credits,
		}
		sch.data[course] = append(sch.data[course], schedule)
	})

	return nil
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