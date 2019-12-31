package database

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"github.com/rothso/isqool/pkg/scrape"
	"google.golang.org/api/googleapi"
)

type BigQuery struct {
	ctx     context.Context
	client  *bigquery.Client
	dataset *bigquery.Dataset
}

func NewBigQuery(projectID, datasetID string) (BigQuery, error) {
	var bq BigQuery

	// Set up BigQuery
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return bq, fmt.Errorf("failed to create client: %v", err)
	}

	dataset := client.Dataset(datasetID)
	if err := dataset.Create(ctx, nil); err != nil {
		if !isDuplicateError(err) {
			return bq, fmt.Errorf("failed to create dataset: %v", err)
		}
	}

	bq = BigQuery{ctx, client, dataset}
	return bq, nil
}

func (bq BigQuery) InsertDepartments(departments []scrape.DeptSchedule) error {
	// Infer the table schema
	schema, err := bigquery.InferSchema(scrape.DeptSchedule{})
	if err != nil {
		return fmt.Errorf("failed to infer schema: %v", err)
	}

	// Get a reference to the table
	table := bq.dataset.Table("departments")
	if err := table.Create(bq.ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		if !isDuplicateError(err) {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	// Create a temp table
	// TODO: Use a different table each time: https://stackoverflow.com/a/51998193/5623874
	newArrivals := bq.dataset.Table("departments_newarrivals")
	if err := newArrivals.Create(bq.ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		if !isDuplicateError(err) {
			return fmt.Errorf("failed to create arrivals table: %v", err)
		}
	}

	// Upload data
	u := newArrivals.Inserter()
	if err := u.Put(bq.ctx, departments); err != nil {
		return fmt.Errorf("failed to insert rows: %v", err)
	}

	// Merge data
	q := bq.client.Query(`
		MERGE isqool.departments t
		USING isqool.departments_newarrivals s
		ON t.course = s.course
		  AND t.term = s.term
		  AND t.crn = s.crn
		  AND (t.instructor = s.instructor
        	OR IFNULL(t.instructor, s.instructor) IS NULL)
		WHEN NOT MATCHED THEN
		  INSERT ROW`)
	if _, err := q.Run(bq.ctx); err != nil { // TODO return status
		panic(fmt.Errorf("failed to execute query: %v", err))
	}

	// Delete temp table
	if err := newArrivals.Delete(bq.ctx); err != nil {
		panic(fmt.Errorf("failed to delete arrivals table: %v", err))
	}

	return nil
}

func isDuplicateError(err error) bool {
	if e, ok := err.(*googleapi.Error); ok {
		return e.Code == 409
	} else {
		return false
	}
}
