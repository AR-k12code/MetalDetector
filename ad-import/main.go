package main

import (
	"context"
	"metaldetector/common"
	"strconv"
	"time"

	"github.com/9072997/jgh"
	"github.com/go-ldap/ldap/v3"
	"github.com/jackc/pgx/v4"
)

type User struct {
	Username      string
	Email         string
	FirstName     string
	MiddleInitial string
	LastName      string
	StudentID     string
	Building      string
	GradYear      string
	Title         string
	CreationDate  string
}

// bind using URL, username, and password from config
func ldapConn() *ldap.Conn {
	conn, err := ldap.DialURL(common.Config.LDAP.URL)
	jgh.PanicOnErr(err)
	err = conn.Bind(common.Config.LDAP.UserDN, common.Config.LDAP.Password)
	jgh.PanicOnErr(err)
	return conn
}

func main() {
	db := common.PGXPool(1)
	defer db.Close()

	conn := ldapConn()
	defer conn.Close()

	query := ldap.NewSearchRequest(
		common.Config.LDAP.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,     // result count limit
		0,     // no timeout
		false, // types only: false
		common.Config.LDAP.Filter,
		[]string{ // attributes
			"sAMAccountName",      // username
			"userPrincipalName",   // email
			"givenName",           // first name
			"initials",            // middle initial
			"sn",                  // last name
			"extensionAttribute1", // student id
			"company",             // building abbreviation
			"department",          // graduation year
			"title",               // "student" or position
			"whenCreated",         // it's what you think
		},
		[]ldap.Control{},
	)

	results, err := conn.SearchWithPaging(
		query,
		uint32(common.Config.LDAP.PageSize),
	)
	jgh.PanicOnErr(err)

	// start a transaction. If anything goes wrong, rollback.
	tx, err := db.Begin(context.TODO())
	jgh.PanicOnErr(err)
	defer tx.Rollback(context.TODO())

	// clear the table
	_, err = tx.Exec(context.TODO(), "TRUNCATE users")
	jgh.PanicOnErr(err)

	for _, entry := range results.Entries {
		user := User{
			Username:      entry.GetAttributeValue("sAMAccountName"),
			Email:         entry.GetAttributeValue("userPrincipalName"),
			FirstName:     entry.GetAttributeValue("givenName"),
			MiddleInitial: entry.GetAttributeValue("initials"),
			LastName:      entry.GetAttributeValue("sn"),
			StudentID:     entry.GetAttributeValue("extensionAttribute1"),
			Building:      entry.GetAttributeValue("company"),
			GradYear:      entry.GetAttributeValue("department"),
			Title:         entry.GetAttributeValue("title"),
			CreationDate:  entry.GetAttributeValue("whenCreated"),
		}
		err = user2db(user, tx)
		jgh.PanicOnErr(err)
	}

	// commit transaction
	err = tx.Commit(context.TODO())
	jgh.PanicOnErr(err)
}

func user2db(user User, tx pgx.Tx) error {
	var studentID *int
	id, err := strconv.Atoi(user.StudentID)
	if err == nil {
		studentID = &id
	}

	var gradYear *int
	year, err := strconv.Atoi(user.GradYear)
	if err == nil {
		gradYear = &year
	}

	var creationDate *time.Time
	d, err := time.Parse(common.Config.LDAP.DateFormat, user.CreationDate)
	if err == nil && user.CreationDate != "" {
		creationDate = &d
	}

	// sometimes a user has a middle name as their middle initial
	if len(user.MiddleInitial) > 1 {
		user.MiddleInitial = user.MiddleInitial[:1]
	}

	_, err = tx.Exec(context.TODO(),
		`
			INSERT INTO users (
				username,
				email,
				first_name,
				middle_initial,
				last_name,
				student_id,
				building,
				graduation_year,
				title,
				creation_date
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10
			)
		`,
		common.EmptyAsNil(user.Username),
		common.EmptyAsNil(user.Email),
		common.EmptyAsNil(user.FirstName),
		common.EmptyAsNil(user.MiddleInitial),
		common.EmptyAsNil(user.LastName),
		studentID,
		common.EmptyAsNil(user.Building),
		gradYear,
		common.EmptyAsNil(user.Title),
		creationDate,
	)
	return err
}
