package main

type Course struct {
	Name       string `db:"name" csv:"course"`
	Term       string `db:"term" csv:"term"`
	Crn        string `db:"crn" csv:"crn"`
	Instructor string `db:"instructor" csv:"instructor"`
}

type Isq struct {
	Enrolled     string `db:"enrolled" csv:"enrolled"`
	Responded    string `db:"responded" csv:"responded"`
	ResponseRate string `db:"response_rate" csv:"response_rate"`
	Percent5     string `db:"percent_5" csv:"percent_5"`
	Percent4     string `db:"percent_4" csv:"percent_4"`
	Percent3     string `db:"percent_3" csv:"percent_3"`
	Percent2     string `db:"percent_2" csv:"percent_2"`
	Percent1     string `db:"percent_1" csv:"percent_1"`
	Rating       string `db:"rating" csv:"rating"`
}

type Grades struct {
	PercentA float32 `db:"percent_a" csv:"A"`
	PercentB float32 `db:"percent_b" csv:"B"`
	PercentC float32 `db:"percent_c" csv:"C"`
	PercentD float32 `db:"percent_d" csv:"D"`
	PercentF float32 `db:"percent_e" csv:"F"`
	Average  string  `db:"average_gpa" csv:"average_gpa"`
}

type Schedule struct {
	StartTime string `db:"start_time" csv:"start_time"`
	Duration  string `db:"duration" csv:"duration"`
	Days      string `db:"days" csv:"days"`
	Building  string `db:"building" csv:"building"`
	Room      string `db:"room" csv:"room"`
	Credits   string `db:"credits" csv:"credits"`
}