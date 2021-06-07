package main

import (
	"context"
	"encoding/json"
	"fmt"
	"metaldetector/common"
	"net/http"
	"net/url"
	"strings"

	"github.com/9072997/jgh"
	"github.com/jackc/pgx/v4"
)

type Student struct {
	StudentID     int    `json:"studentId"`
	Email         string `json:"email"`
	FirstName     string `json:"firstName"`
	MiddleInitial string `json:"middleInitial"`
	LastName      string `json:"lastName"`
	Nickname      string `json:"nickname"`
	Building      string `json:"building"`
	Grade         string `json:"grade"`
	Status        string `json:"status"`
}

func main() {
	db := common.PGXPool(1)
	defer db.Close()

	u, err := url.Parse(common.Config.ESchool.ReportURL)
	jgh.PanicOnErr(err)
	resp, err := http.DefaultClient.Do(&http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"X-API-Key": []string{common.Config.ESchool.APIKey},
			"Accept":    []string{"application/json"},
		},
	})
	jgh.PanicOnErr(err)

	decoder := json.NewDecoder(resp.Body)

	// discard open bracket
	_, err = decoder.Token()
	jgh.PanicOnErr(err)

	// start a transaction. If anything goes wrong, rollback.
	tx, err := db.Begin(context.TODO())
	jgh.PanicOnErr(err)
	defer tx.Rollback(context.TODO())

	// clear the table
	_, err = tx.Exec(context.TODO(), "TRUNCATE students")
	jgh.PanicOnErr(err)

	// while the array contains values
	for decoder.More() {
		// decode a single student
		var s Student
		err = decoder.Decode(&s)
		jgh.PanicOnErr(err)

		err = student2db(s, tx)
		jgh.PanicOnErr(err)
	}

	// discard closing bracket
	_, err = decoder.Token()
	jgh.PanicOnErr(err)

	// commit transaction
	err = tx.Commit(context.TODO())
	jgh.PanicOnErr(err)
}

func student2db(s Student, tx pgx.Tx) error {
	switch strings.ToUpper(s.Grade) {
	case "PK":
		s.Grade = "PRE-K"
	case "KF":
		s.Grade = "KINDERGARTEN"
	case "01":
		s.Grade = "1"
	case "02":
		s.Grade = "2"
	case "03":
		s.Grade = "3"
	case "04":
		s.Grade = "4"
	case "05":
		s.Grade = "5"
	case "06":
		s.Grade = "6"
	case "07":
		s.Grade = "7"
	case "08":
		s.Grade = "8"
	case "09":
		s.Grade = "9"
	case "10":
		s.Grade = "10"
	case "11":
		s.Grade = "11"
	case "12":
		s.Grade = "12"
	case "SS":
		s.Grade = "SUPER-SENIOR"
	case "GG":
		s.Grade = "GRADUATED"
	case "SM":
		s.Grade = "SM"
	default:
		return fmt.Errorf("invalid student grade: %s", s.Grade)
	}

	switch strings.ToUpper(s.Status) {
	case "A":
		s.Status = "ACTIVE"
	case "G":
		s.Status = "GRADUATED"
	case "I":
		s.Status = "INACTIVE"
	case "P":
		s.Status = "PRE-REGISTERED"
	default:
		return fmt.Errorf("invalid student status: %s", s.Status)
	}

	_, err := tx.Exec(context.TODO(),
		`
			INSERT INTO students (
				student_id,
				email,
				first_name,
				middle_initial,
				last_name,
				nickname,
				building,
				grade,
				status
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9
			)
		`,
		s.StudentID,
		common.EmptyAsNil(s.Email),
		common.EmptyAsNil(s.FirstName),
		common.EmptyAsNil(s.MiddleInitial),
		common.EmptyAsNil(s.LastName),
		common.EmptyAsNil(s.Nickname),
		common.EmptyAsNil(s.Building),
		common.EmptyAsNil(s.Grade),
		common.EmptyAsNil(s.Status),
	)
	return err
}
