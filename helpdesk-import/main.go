package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"metaldetector/common"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/9072997/jgh"
	"github.com/gammazero/workerpool"
	"github.com/jackc/pgx/v4"
)

func main() {
	// one connection for reading serial number, one for inserts
	db := common.PGXPool(2)
	defer db.Close()

	// start a transaction. If anything goes wrong, rollback.
	tx, err := db.Begin()
	jgh.PanicOnErr(err)
	defer tx.Rollback()

	// clear the table
	_, err = tx.Exec("TRUNCATE assets")
	jgh.PanicOnErr(err)

	// Fetching info from the helpdesk is high latency, so run multiple
	// requests in parallell.
	fetchWP := workerpool.New(common.Config.Helpdesk.MaxConcurentRequests)

	// transactions can only process a single query at a time, so we run a
	// single database-inserter thread. NOTE: MaxConcurentRequests is just
	// a reasonable value for a buffer size here. Any value should be
	// acceptable
	var insertWG sync.WaitGroup
	assets := make(chan Asset, common.Config.Helpdesk.MaxConcurentRequests)
	go func() {
		for a := range assets {
			err = asset2db(a, tx)
			jgh.PanicOnErr(err)
			insertWG.Done()
		}
	}()

	// we will only imports assets that have serial numbers listed in
	// Google Admin, so get a list of serial numbers in Google Admin
	rows, err := db.Query("SELECT DISTINCT serial FROM chromebooks")
	jgh.PanicOnErr(err)

	// all the REST requests will be really loud, so disable logging
	jgh.Logger = log.New(ioutil.Discard, "", 0)

	// for each serial number in the chromebooks table, get an Asset object
	// and send it to the channel
	for rows.Next() {
		var serial string
		err = rows.Scan(&serial)
		jgh.PanicOnErr(err)

		fetchWP.Submit(func() {
			a, err := asset(serial)
			if err != nil {
				log.Printf("Error getting asset %s: %s", serial, err)
				return
			}

			insertWG.Add(1)
			assets <- a
		})
	}
	jgh.PanicOnErr(rows.Err())
	rows.Close()

	// wait for all threads to finish
	wpProgress(fetchWP, time.Second)
	fetchWP.StopWait()
	insertWG.Wait()

	// commit transaction
	err = tx.Commit()
	jgh.PanicOnErr(err)
}

func wpProgress(wp *workerpool.WorkerPool, interval time.Duration) {
	originalSize := wp.WaitingQueueSize()
	ticker := time.NewTicker(interval).C
	for {
		size := wp.WaitingQueueSize()
		done := originalSize - size
		percent := done * 100 / originalSize
		fmt.Printf("%02d%% (%d/%d)\n", percent, size, originalSize)

		if size == 0 {
			return
		}

		<-ticker
	}
}

func asset2db(a Asset, tx *pgx.Tx) error {
	var assetNumber *int
	an, err := strconv.Atoi(a.AssetNumber)
	if err == nil {
		assetNumber = &an
	}

	var clients string
	for _, client := range a.Clients {
		clients += client.Email + ","
	}
	clients = strings.TrimSuffix(clients, ",")

	// translations for enum type in DB
	switch a.AssetStatus.Name {
	case "Deployed":
		a.AssetStatus.Name = "DEPLOYED"
	case "Surplus/Retired":
		a.AssetStatus.Name = "RETIRED"
	case "Stolen":
		a.AssetStatus.Name = "STOLEN"
	case "Surplus/Broken":
		a.AssetStatus.Name = "BROKEN"
	case "Unassigned":
		a.AssetStatus.Name = "UNASSIGNED"
	case "Disabled":
		a.AssetStatus.Name = "DISABLED"
	case "Lost":
		a.AssetStatus.Name = "LOST"
	}

	_, err = tx.Exec(
		`
			INSERT INTO assets (
				serial,
				asset_number,
				status,
				location,
				room,
				model,
				client,
				notes
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8
			)
		`,
		common.EmptyAsNil(a.SerialNumber),
		assetNumber,
		common.EmptyAsNil(a.AssetStatus.Name),
		common.EmptyAsNil(a.Location.Name),
		common.EmptyAsNil(a.Room.RoomName),
		common.EmptyAsNil(a.Model.Name),
		common.EmptyAsNil(clients),
		common.EmptyAsNil(a.Notes),
	)
	return err
}
