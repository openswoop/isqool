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

type Course struct {
	ID         uint64 `db:"id,primarykey,autoincrement" csv:"-"`
	Name       string `db:"name" csv:"course"`
	Term       string `db:"term" csv:"term"`
	Crn        string `db:"crn" csv:"crn"`
	Instructor string `db:"instructor" csv:"instructor"`
}

type IsqData struct {
	ID           uint64 `db:"id,primarykey,autoincrement" csv:"-"`
	CourseID     uint64 `db:"course_data" csv:"-"` // TODO mark db foreign key
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

type GradeDist struct {
	ID       uint64  `db:"id,primarykey,autoincrement" csv:"-"`
	CourseID uint64  `db:"course_id" csv:"-"` // TODO mark db foreign key
	PercentA float32 `db:"percent_a" csv:"A"`
	PercentB float32 `db:"percent_b" csv:"B"`
	PercentC float32 `db:"percent_c" csv:"C"`
	PercentD float32 `db:"percent_d" csv:"D"`
	PercentF float32 `db:"percent_e" csv:"F"`
	Average  string  `db:"average_gpa" csv:"average_gpa"`
}

type Schedule struct {
	ID        uint64 `db:"id,primarykey,autoincrement" csv:"-"`
	CourseID  uint64 `db:"course_id" csv:"-"` // TODO mark db foreign key
	StartTime string `db:"start_time" csv:"start_time"`
	Duration  string `db:"duration" csv:"duration"`
	Days      string `db:"days" csv:"days"`
	Building  string `db:"building" csv:"building"`
	Room      string `db:"room" csv:"room"`
	Credits   string `db:"credits" csv:"credits"`
}

type Record struct {
	Course
	IsqData
	GradeDist
	Schedule
}

var (
	cacheDir = "./.webcache"
	dbFile   = "isqool.db"
)

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
		err := tx.Insert(&record.Course)
		if err != nil {
			// TODO: gracefully skip courses already in the DB
			return err
		}
		// Assign foreign keys
		courseId := record.Course.ID
		record.IsqData.CourseID = courseId
		record.GradeDist.CourseID = courseId
		record.Schedule.CourseID = courseId
		err = tx.Insert(&record.IsqData, &record.GradeDist, &record.Schedule)
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
	dbmap.AddTableWithName(IsqData{}, "isq")
	dbmap.AddTableWithName(GradeDist{}, "grades")
	dbmap.AddTableWithName(Schedule{}, "sections")
	err := dbmap.CreateTablesIfNotExists() // TODO: use a migration tool
	if err != nil {
		panic(err)
	}

	return dbmap
}

func getCourseRecords(course string) (map[Course]IsqData, map[Course]GradeDist, error) {
	col := colly.NewCollector()
	col.CacheDir = cacheDir

	// Map by courseKey => data so we can later join the data sets
	isqData := make(map[Course]IsqData)
	distData := make(map[Course]GradeDist)

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

			id := Course{
				Name:       course,
				Term:       cells.Eq(0).Text(),
				Crn:        cells.Eq(1).Text(),
				Instructor: strings.TrimSpace(cells.Eq(2).Text()),
			}
			dist := GradeDist{
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

func timetrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}