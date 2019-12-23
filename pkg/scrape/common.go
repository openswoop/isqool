package scrape

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

const bannerUrl = "https://banner.unf.edu/pls/nfpo/"

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

func CollectScheduleParams(isqs []Isq, grades []Grades) []ScheduleParams {
	// Collect all the courses
	var courses []Course
	for _, isq := range isqs {
		courses = append(courses, isq.Course)
	}
	for _, grade := range grades {
		courses = append(courses, grade.Course)
	}

	// Find all the unique course/term combinations we need to query
	var seen = make(map[ScheduleParams]bool)
	var params []ScheduleParams

	for _, course := range courses {
		subject := course.Name[0:3]
		courseNumber, err := strconv.Atoi(course.Name[3:])
		if err != nil {
			continue
		}
		term, err := TermToId(course.Term)
		if err != nil {
			continue
		}

		param := ScheduleParams{subject, courseNumber, term}
		if _, found := seen[param]; !found {
			params = append(params, param)
			seen[param] = true
		}
	}
	return params
}

func getLastName(instructor string) string {
	instructorR := regexp.MustCompile(`\s((?:de |Von )?[\w-']+)(?: \(P\)(?:,.*)?)?$`)
	return instructorR.FindStringSubmatch(instructor)[1]
}
