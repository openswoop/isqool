package main

import (
	"os"
)

type Dataset map[Course][]Feature

func main() {
	course := os.Args[1]

	// We want the following features
	features := []Feature{
		Isq{},
		Grades{},
		Schedule{},
	}

	// Register the scrapers
	resolvers := ResolverMap{}
	resolvers.Register(Isq{}, ResolveIsq)
	resolvers.Register(Grades{}, ResolveGrades)
	resolvers.Register(Schedule{}, ResolveSchedule)

	// Ready, set, go!
	data, err := resolvers.Resolve(course, features)
	if err != nil {
		panic(err)
	}

	// Ensure these models are in the DB
	// Open up a transaction
	//var db Database
	//db.PersistEntity(data)
	data.Persist(NoopDb{})
}
