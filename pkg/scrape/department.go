package scrape

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"strconv"
	"strings"
	"time"
)

type Meeting struct {
	BeginDate time.Time
	EndDate   time.Time
	Days      string
	BeginTime time.Time
	EndTime   time.Time
	Type      string
	Building  string
	Room      int
}

type DeptSchedule struct {
	Course
	Status      string
	Title       string
	InstructorN int
	Credits     int
	PartOfTerm  string
	Meetings    []Meeting
	Campus      string
	WaitCount   int
	Approval    string
}

func GetDepartment(c *colly.Collector, term string, deptId int) ([]DeptSchedule, error) {
	var department []DeptSchedule

	// Collect the data for each course listing in the department and term
	c.OnHTML(".pagebodydiv > .datadisplaytable", func(e *colly.HTMLElement) {
		// Select all rows after the header row
		rows := e.DOM.Find("tr:nth-child(n+2)")

		rows.Each(func(_ int, s *goquery.Selection) {
			cells := s.Find("td")

			// If this row is a continuation of the previous row
			continuation := s.Find("td[colspan]").Size() != 0

			// Shift the cells 5 to the left if it's a continuation
			offset := 0
			if continuation {
				offset = 4
			}

			// Extract the begin and end date
			var beginDate, endDate time.Time
			if strings.TrimSpace(cells.Eq(6-offset).Text()) != "" {
				currYear := strings.Split(term, " ")[1]
				beginDate, _ = time.Parse("01-02-2006", cells.Eq(6-offset).Text()+"-"+currYear)
				endDate, _ = time.Parse("01-02-2006", cells.Eq(7-offset).Text()+"-"+currYear)
			}

			// Extract the begin and end time
			var beginTime, endTime time.Time
			if strings.TrimSpace(cells.Eq(9-offset).Text()) != "" {
				beginTime, _ = time.Parse("03:04PM", cells.Eq(9-offset).Text())
				endTime, _ = time.Parse("03:04PM", cells.Eq(10-offset).Text())
			}

			// Extract the room number
			var room int
			if strings.TrimSpace(cells.Eq(13-offset).Text()) != "" {
				room = atoi(cells.Eq(13 - offset).Text())
			}

			meeting := Meeting{
				BeginDate: beginDate,
				EndDate:   endDate,
				Days:      strings.Replace(cells.Eq(8-offset).Text(), " ", "", -1),
				BeginTime: beginTime,
				EndTime:   endTime,
				Type:      strings.TrimSpace(cells.Eq(11 - offset).Text()),
				Building:  strings.TrimSpace(cells.Eq(12 - offset).Text()),
				Room:      room,
			}

			// If this is not a continuation of a previous row
			if !continuation {
				// Extract the instructor's n#
				instructor := strings.TrimSpace(cells.Eq(17).Text())
				var instructorN int
				if instructor != "" {
					link, _ := cells.Eq(17).Find("a").First().Attr("href")
					instructorN = atoi(strings.Split(link, "=N")[1])
				}

				// Extract the number of credits the course is worth
				creditsStr := strings.TrimSpace(cells.Eq(4).Text())
				credits := atoi(strings.Split("0"+creditsStr, ".")[0])

				// If meeting time is blank (class got cancelled), don't include it
				var meetings []Meeting = nil
				if (Meeting{}) != meeting {
					meetings = []Meeting{meeting}
				}

				course := Course{
					Name:       strings.TrimSpace(cells.Eq(2).Text()),
					Term:       term,
					Crn:        atoi(cells.Eq(1).Text()),
					Instructor: instructor,
				}
				deptSchedule := DeptSchedule{
					Course:      course,
					Status:      strings.TrimSpace(cells.Eq(0).Text()),
					Title:       strings.TrimSpace(cells.Eq(3).Text()),
					InstructorN: instructorN,
					Credits:     credits,
					PartOfTerm:  strings.Split(cells.Eq(5).Text(), " - ")[0],
					Meetings:    meetings,
					Campus:      strings.TrimSpace(cells.Eq(14).Text()),
					WaitCount:   atoi(cells.Eq(15).Text()),
					Approval:    strings.TrimSpace(cells.Eq(16).Text()),
				}
				department = append(department, deptSchedule)
			} else {
				// Attach to previous row
				parent := department[len(department)-1]
				parent.Meetings = append(parent.Meetings, meeting)
				department[len(department)-1] = parent
			}
		})
	})

	termId, _ := TermToId(term)
	return department, c.Post(bannerUrl+"wksfwbs.p_dept_schd", map[string]string{
		"pv_term":   strconv.Itoa(termId),
		"pv_dept":   strconv.Itoa(deptId),
		"pv_ptrm":   "",
		"pv_campus": "",
		"pv_sub":    "Submit",
	})
}
