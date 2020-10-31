package scrape

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Schedule struct {
	StartTime string `db:"start_time" csv:"start_time"`
	Duration  string `db:"duration" csv:"duration"`
	Days      string `db:"days" csv:"days"`
	Building  string `db:"building" csv:"building"`
	Room      string `db:"room" csv:"room"`
	Credits   string `db:"credits" csv:"credits"`
	Title     string `db:"title" csv:"title"`
}

type CourseSchedule struct {
	Course
	Schedule
}

type ScheduleParams struct {
	Subject      string
	CourseNumber string
	TermId       int
}

func GetSchedules(c *colly.Collector, params []ScheduleParams) ([]CourseSchedule, error) {
	var schedules []CourseSchedule

	// Collect the schedules
	c.OnHTML("body", func(e *colly.HTMLElement) {
		tables := e.DOM.Find("table.datadisplaytable:nth-child(5) > tbody > tr:nth-child(even)")
		term := strings.TrimSpace(e.DOM.Find(".staticheaders").Contents().Eq(0).Text())

		tables.Each(func(_ int, s *goquery.Selection) {
			// TODO: store all the scheduled meeting times (don't filter)
			rows := s.Find("td table.datadisplaytable tr").FilterFunction(func(_ int, s *goquery.Selection) bool {
				// Ignore any laboratory information, for now
				// TODO: store class type ("class" or "laboratory") as Schedule field
				return strings.Contains(s.Find("td").First().Text(), "Class")
			})

			// Unique key for the map
			headerData := strings.Split(s.Prev().Text(), " - ")

			course := Course{
				Name: strings.Replace(headerData[2], " ", "", 1),
				Term: term,
				Crn:  atoi(headerData[1]),
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
			course.Instructor = nullString(getLastName(data.Last().Text()))

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
			if locationText != "Online" && locationText != "Off Main Campus" && locationText != "TBA" {
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

			// Extract the official name of the course
			titleR := regexp.MustCompile(`^.*(?:\(\w+\)|H-)\s*`)
			title := titleR.ReplaceAllString(strings.TrimSpace(headerData[0]), "")

			schedule := Schedule{
				StartTime: startTime,
				Duration:  duration,
				Days:      days,
				Building:  building,
				Room:      room,
				Credits:   credits,
				Title:     title,
			}
			schedules = append(schedules, CourseSchedule{course, schedule})
		})
	})

	var err error
	for _, p := range params {
		url := fmt.Sprintf(
			"%vbwckctlg.p_disp_listcrse?schd_in=&subj_in=%v&crse_in=%v&term_in=%d",
			bannerUrl, p.Subject, p.CourseNumber, p.TermId)
		err = c.Visit(url)
		if err != nil {
			return nil, err
		}
	}

	return schedules, nil
}
