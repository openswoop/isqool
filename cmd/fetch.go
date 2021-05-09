package cmd

import (
	"github.com/rothso/isqool/pkg/database"
	"github.com/rothso/isqool/pkg/report"
	"github.com/rothso/isqool/pkg/scrape"
	"log"
	"os"
	"regexp"

	"github.com/spf13/cobra"
)

var dbFile = "/isqool/isqool.db"

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch [course|professor]",
	Short: "Scrape summary data to a CSV file",
	Long: `Given a course name or professor's N# this command will output
a CSV file from the historical course data available. The
results will also be inserted into a local SQLite database.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0] // COT3100 or N00474503 etc.
		isProfessor, _ := regexp.MatchString("N\\d{8}", name)

		// Scrape the data
		isqs, grades, err := scrape.GetIsqAndGrades(c.Clone(), name, isProfessor)
		if err != nil {
			panic(err)
		}
		params := scrape.CollectScheduleParams(isqs, grades)
		schedules, err := scrape.GetSchedules(c.Clone(), params)
		if err != nil {
			panic(err)
		}
		log.Println("Found", len(schedules), "records")

		// Save all the data to the database
		userCacheDir, _ := os.UserCacheDir()
		sqlite := database.NewSqlite(userCacheDir + dbFile)
		if err := sqlite.SaveIsqs(isqs); err != nil {
			panic(err)
		}
		if err := sqlite.SaveGrades(grades); err != nil {
			panic(err)
		}
		if err := sqlite.SaveSchedules(schedules); err != nil {
			panic(err)
		}
		_ = sqlite.Close()
		log.Println("Saved to database", dbFile)

		// Write to CSV
		err = report.WriteCourse(name, report.CourseInput{
			Isqs:      isqs,
			Grades:    grades,
			Schedules: schedules,
		})
		if err != nil {
			panic(err)
		}
		log.Println("Wrote to file", name+".csv")
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}
