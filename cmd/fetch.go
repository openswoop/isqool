package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/openswoop/isqool/pkg/database"
	"github.com/openswoop/isqool/pkg/report"
	"github.com/openswoop/isqool/pkg/scrape"

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
		// Professor name is present
		if !isProfessor && len(name) > 7 {
			// Adds name to search term Replace space with %20
			url := fmt.Sprintf("https://webapps.unf.edu/faculty/bio/api/v1/faculty?searchLimit=1&searchTerm=%v", strings.ReplaceAll(name, " ", "%20")) 
			log.Println(url)
			response, error := http.Get(url)
			if error != nil {
				panic(error)
			}
			defer response.Body.Close()
			body, error := io.ReadAll(response.Body)
			if error != nil {
				panic(error)
			}
			re := regexp.MustCompile("N\\d{8}")
			name = re.FindString(string(body))
			isProfessor = true
		}
		log.Println(name)
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
