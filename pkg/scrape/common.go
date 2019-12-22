package scrape

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// TermToId takes a term string like "Fall 2017" and determines its
// corresponding id (e.g: 201780)
func TermToId(term string) (int, error) {
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

func getLastName(instructor string) string {
	instructorR := regexp.MustCompile(`\s((?:de |Von )?[\w-']+)(?: \(P\)(?:,.*)?)?$`)
	return instructorR.FindStringSubmatch(instructor)[1]
}
