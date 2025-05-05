package cmd

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"fmt"
	"github.com/openswoop/isqool/pkg/database"
	"github.com/openswoop/isqool/pkg/report"
	"github.com/openswoop/isqool/pkg/scrape"
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

const (
	projectID = "syllabank-4e5b9"
	datasetID = "isqool"
	topicID   = "department-refreshed"
)

var dryRun bool
var debug bool

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Scrape departmental data to BigQuery",
	Long: `This command takes a department ID and term (such as "Spring 2020")
and scrapes the ISQs, grades, and schedules of the courses offered.`,
	Run: func(cmd *cobra.Command, args []string) {
		deptId, _ := strconv.Atoi(args[0]) // e.g. 6502
		seedTerm := args[1]                // e.g. Spring 2020

		// Scrape the first term as a starting point
		initialDept, err := scrape.GetDepartment(c, seedTerm, deptId)
		if err != nil {
			panic(err)
		}

		// If the debug flag is set, output the CSV and exit early
		if debug {
			err := report.WriteDepartment(fmt.Sprintf("%d_%s", deptId, seedTerm), initialDept)
			if err != nil {
				panic(err)
			}
			return
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
		if !dryRun {
			if err := bq.InsertDepartments(deptTable, deptId, seedTerm); err != nil {
				panic(fmt.Errorf("failed to insert department schedule: %v", err))
			}
			if err := bq.InsertISQs(isqTable); err != nil {
				panic(fmt.Errorf("failed to insert isqs: %v", err))
			}
			if err := bq.InsertGrades(gradesTable); err != nil {
				panic(fmt.Errorf("failed to insert grades: %v", err))
			}
		} else {
			fmt.Println("Dry run: data will not be inserted")
		}

		// Connect to PubSub
		ctx := context.Background()
		client, err := pubsub.NewClient(ctx, projectID)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}

		msg, err := json.Marshal(struct {
			DepartmentId int `json:"departmentId"`
		}{deptId})
		if err != nil {
			log.Fatalf("Failed to create message: %v", err)
		}

		// Publish an event
		topic := client.Topic(topicID)
		res := topic.Publish(ctx, &pubsub.Message{Data: msg})
		if _, err := res.Get(ctx); err != nil {
			log.Fatalf("Failed to publish message: %v", err)
		}

		fmt.Println("Done.")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands:
	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Run without modifying the database (default: false)")

	// Cobra supports local flags which will only run when this command
	// is called directly:
	syncCmd.Flags().BoolVar(&debug, "debug", false, "Dump the departmental summary as a CSV (default: false)")
}
