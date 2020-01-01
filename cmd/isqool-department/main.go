package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/database"
	"github.com/rothso/isqool/pkg/scrape"
	"os"
)

var (
	projectID = "syllabank-4e5b9"
	datasetID = "isqool"
)

func main() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}

	// Set up colly
	c := colly.NewCollector()
	c.CacheDir = userCacheDir + "/isqool/web-cache"
	c.AllowURLRevisit = true

	seedTerm := "Spring 2020"
	deptId := 6502

	// Scrape the first term as a starting point
	initialDept, err := scrape.GetDepartment(c, seedTerm, deptId)
	if err != nil {
		panic(err)
	}

	seen := make(map[string]bool)
	var courses []string
	for _, row := range initialDept {
		if _, found := seen[row.Name]; !found {
			courses = append(courses, row.Name)
			seen[row.Name] = true
		}
	}

	// Scrape all the courses offered that term
	var isqTable []scrape.CourseIsq
	var gradesTable []scrape.CourseGrades
	for _, course := range courses {
		isqs, grades, err := scrape.GetIsqAndGrades(c.Clone(), course, false)
		if err != nil {
			panic(err)
		}
		isqTable = append(isqTable, isqs...)
		gradesTable = append(gradesTable, grades...)
	}

	seen = make(map[string]bool)
	var terms []string
	for _, row := range isqTable {
		if _, found := seen[row.Term]; !found && row.Term != seedTerm {
			terms = append(terms, row.Term)
			seen[row.Term] = true
		}
	}

	// Scrape all the terms those courses were offered in
	deptTable := initialDept
	for _, term := range terms {
		dept, err := scrape.GetDepartment(c, term, deptId)
		if err != nil {
			panic(err)
		}
		deptTable = append(deptTable, dept...)
	}

	// Connect to BigQuery
	bq, err := database.NewBigQuery(projectID, datasetID)
	if err != nil {
		panic(fmt.Errorf("failed to connect to bigquery: %v", err))
	}

	// Insert (merge) the department schedules, isqs, and grades
	if err := bq.InsertDepartments(deptTable); err != nil {
		panic(fmt.Errorf("failed to insert department schedule: %v", err))
	}
	if err := bq.InsertISQs(isqTable); err != nil {
		panic(fmt.Errorf("failed to insert isqs: %v", err))
	}
	if err := bq.InsertGrades(gradesTable); err != nil {
		panic(fmt.Errorf("failed to insert grades: %v", err))
	}

	fmt.Println("Done.")
}
