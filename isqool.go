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
	"database/sql"
	"github.com/go-gorp/gorp"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Record struct {
	Course
	Isq
	Grades
	Schedule
}

func main_old() {
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

	// Merge the ISQ records, grade distributions, and schedule details by Course,
	// only keeping records that have all three parts (i.e. the intersecting set)
	records := make([]Record, 0)
	log.Println("Compiling records")
	// TODO: iterate in order, to make CSVs and database easier to scan as a human
	for id, isq := range isqData {
		if dist, ok := distData[id]; ok {
			if schedule, ok := scheduleData[id]; ok {
				records = append(records, Record{id, isq, dist, schedule})
			} else {
				// If this happens, the schedule parser is b0rked or incomplete
				class := id.Term + " " + id.Crn + " " + id.Instructor
				log.Println("Unable to match schedule data for " + class)
			}
		} else {
			// TODO handle labs? (they don't have grade data)
			log.Println("Omitting", id, "(no grades)")
		}
	}
	log.Println("Found", len(records), "records")

	// Write to CSV file
	log.Println("Saving to", course+".csv")
	if err := writeCsv(course, records); err != nil {
		panic(err)
	}

	// Persist to SQLite3 database
	log.Println("Updating local database")
	if err := persistDb(records); err != nil {
		panic(err)
	}
}

func writeCsv(course string, data interface{}) error {
	file, err := os.Create(course + ".csv")
	if err != nil {
		return err
	}
	defer file.Close()
	return gocsv.MarshalFile(data, file)
}

func persistDb(records []Record) error {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}
	dbmap := initDbMap(db)
	defer db.Close()

	// Wrap inserts in a transaction for the performance gain
	tx, err := dbmap.Begin()
	if err != nil {
		return err
	}
	for _, record := range records {
		course := &CourseEntity{Course: record.Course}
		err := tx.Insert(course)
		if err != nil {
			// TODO: gracefully skip courses already in the DB
			return err
		}
		// Assign foreign keys
		courseKey := CourseKey{course.ID}
		isq := &IsqEntity{CourseKey: courseKey, Isq: record.Isq}
		grades := &GradesEntity{CourseKey: courseKey, Grades: record.Grades}
		schedule := &ScheduleEntity{CourseKey: courseKey, Schedule: record.Schedule}
		err = tx.Insert(isq, grades, schedule)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()

	return err
}

func initDbMap(db *sql.DB) *gorp.DbMap {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	//dbmap.TraceOn("[gorp]", log.New(os.Stdout, "isqool: ", log.Lmicroseconds))
	dbmap.AddTableWithName(Course{}, "courses").SetUniqueTogether("Crn", "Term", "Instructor", "Name")
	dbmap.AddTableWithName(IsqEntity{}, "isq")
	dbmap.AddTableWithName(GradesEntity{}, "grades")
	dbmap.AddTableWithName(ScheduleEntity{}, "sections")
	err := dbmap.CreateTablesIfNotExists() // TODO: use a migration tool
	if err != nil {
		panic(err)
	}

	return dbmap
}

func getCourseRecords(course string) (map[Course]Isq, map[Course]Grades, error) {
	col := colly.NewCollector()
	col.CacheDir = cacheDir

	// Map by courseKey => data so we can later join the data sets
	isqData := make(map[Course]Isq)
	distData := make(map[Course]Grades)

	// Download the "Instructional Satisfaction Questionnaire" table
	col.OnHTML("table.datadisplaytable:nth-child(9)", func(e *colly.HTMLElement) {
		// Read each row sequentially, skipping the two header rows
		e.DOM.Find("tr:nth-child(n+3)").Each(func(i int, s *goquery.Selection) {
			cells := s.Find("td")
			id := Course{
				Name:       course,
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

			id := Course{
				Name:       course,
				Term:       cells.Eq(0).Text(),
				Crn:        cells.Eq(1).Text(),
				Instructor: strings.TrimSpace(cells.Eq(2).Text()),
			}
			dist := Grades{
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

func getScheduleDetails(course string, terms []string) (map[Course]Schedule, error) {
	col := colly.NewCollector()
	col.CacheDir = cacheDir

	subject := course[0:3]
	courseId := course[3:]
	urlBase := "https://banner.unf.edu/pls/nfpo/bwckctlg.p_disp_listcrse?schd_in=" +
		"&subj_in=" + subject + "&crse_in=" + courseId + "&term_in="

	schedules := make(map[Course]Schedule)

	selector := "table.datadisplaytable:nth-child(5) > tbody > tr:nth-child(even)"
	col.OnHTML(selector, func(e *colly.HTMLElement) {
		rows := e.DOM.Find("td table.datadisplaytable tr")
		matches := rows.FilterFunction(func(i int, s *goquery.Selection) bool {
			// Ignore any laboratory information, for now
			return strings.Contains(s.Find("td").First().Text(), "Class")
		})

		// Unique key for the map
		id := Course{
			Name: course,
			Term: e.Request.Ctx.Get("term"),
			Crn:  strings.Split(e.DOM.Prev().Text(), " - ")[1],
		}

		if matches.Size() == 0 {
			// Some classes are "hybrid" classes that only meet on certain weeks of the
			// month, which require special parsing. Will be implemented soon.
			log.Println("Warning:", id, "has a hybrid schedule; omitting scheduling data")
			return; // TODO: collapse all hybrid information into one record
		} else if matches.Size() > 1 {
			// Some classes have an extra meeting on Friday at a different time than the
			// other meetings, which cannot be represented in the current CSV structure.
			log.Println("Warning:", id, "met at uneven times; omitting additional blocks")
		}

		data := matches.First().Find("td")

		// Extract the instructor's last name
		instructorR := regexp.MustCompile(`\s((?:de |Von )?[\w-]+) \(P\)`)
		instructor := instructorR.FindStringSubmatch(data.Last().Text())[1]
		id.Instructor = instructor

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

		schedule := Schedule{
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