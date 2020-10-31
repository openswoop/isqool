package cmd

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/spf13/cobra"
	"os"
)

var c *colly.Collector

var cacheDir = "/isqool/web-cache"
var noCache bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "isqool",
	Short: "A tool for scraping historical course data from UNF",
	Long: `Scrapes historical course data from UNF into a format suitable for
analysis. Given a course code, professor's N#, or department ID, this
application can generate a CSV file or send the results to BigQuery.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initColly)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Bypass the web cache (default: false)")
}

func initColly() {
	c = colly.NewCollector()
	if !noCache {
		userCacheDir, _ := os.UserCacheDir()
		c.CacheDir = userCacheDir + cacheDir
	}
}
