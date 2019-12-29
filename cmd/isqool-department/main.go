package main

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/rothso/isqool/pkg/scrape"
	"google.golang.org/api/googleapi"
)

var (
	projectID = "syllabank-4e5b9"
	datasetID = "isqool"
)

func main() {
	// Set up colly
	c := colly.NewCollector()
	c.AllowURLRevisit = true

	dept, err := scrape.GetDepartment(c, "Spring 2019", 6502)
	if err != nil {
		panic(err)
	}

	// Set up BigQuery
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		panic(fmt.Errorf("failed to create client: %v", err))
	}

	dataset := client.Dataset(datasetID)
	if err := dataset.Create(ctx, nil); err != nil {
		if !isDuplicateError(err) {
			panic(fmt.Errorf("failed to create dataset: %v", err))
		}
	}

	schema, err := bigquery.InferSchema(scrape.DeptSchedule{})
	if err != nil {
		panic(fmt.Errorf("failed to infer schema: %v", err))
	}

	table := dataset.Table("departments")
	if err := table.Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		if !isDuplicateError(err) {
			panic(fmt.Errorf("failed to create table: %v", err))
		}
	}

	// Create a temp table
	newArrivals := dataset.Table("departments_newarrivals")
	if err := newArrivals.Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		if !isDuplicateError(err) {
			panic(fmt.Errorf("failed to create arrivals table: %v", err))
		}
	}

	// Upload data
	u := newArrivals.Inserter()
	if err := u.Put(ctx, dept); err != nil {
		panic(fmt.Errorf("failed to insert rows: %v", err))
	}

	// Merge data
	q := client.Query(`
		MERGE isqool.departments t
		USING isqool.departments_newarrivals s
		ON t.course = s.course
		  AND t.term = s.term
		  AND t.crn = s.crn
		  AND (t.instructor = s.instructor
        	OR IFNULL(t.instructor, s.instructor) IS NULL)
		WHEN NOT MATCHED THEN
		  INSERT ROW`)
	if _, err := q.Run(ctx); err != nil {
		panic(fmt.Errorf("failed to execute query: %v", err))
	}

	// Delete temp table
	if err := newArrivals.Delete(ctx); err != nil {
		panic(fmt.Errorf("failed to delete arrivals table: %v", err))
	}

	fmt.Println("Done.")
}

func isDuplicateError(err error) bool {
	if e, ok := err.(*googleapi.Error); ok {
		return e.Code == 409
	} else {
		return false
	}
}
