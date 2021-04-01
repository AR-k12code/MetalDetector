package common

import (
	"github.com/9072997/jgh"
	"github.com/jackc/pgx"
)

// if a string is empty, return nil, else return pointer to string
func EmptyAsNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// return a pgx ConnPool. Set maxConnections to 0 to use value from config.
func PGXPool(maxConnections int) *pgx.ConnPool {
	config := Config.PgSQL
	if maxConnections > 0 {
		config.MaxConnections = maxConnections
	}
	pool, err := pgx.NewConnPool(config)
	jgh.PanicOnErr(err)
	return pool
}
