package main

import (
	"fmt"
	"metaldetector/common"
	"time"

	"github.com/9072997/jgh"
	admin "google.golang.org/api/admin/directory/v1"
)

func main() {
	db := common.PGXPool(2)
	defer db.Close()

	// get authorized google admin client (may prompt for oauth flow)
	ga := common.GAdmin()

	// get predictions that are not already reflected in the admin console
	rows, err := db.Query(`
		SELECT
			predictions.device_id,
			predictions.user
		FROM predictions
		LEFT JOIN chromebooks
			ON predictions.device_id = chromebooks.device_id
		WHERE
			predictions.user <> chromebooks.asset_id OR
			chromebooks.asset_id IS NULL
	`)
	jgh.PanicOnErr(err)
	defer rows.Close()

	for rows.Next() {
		var deviceID, user string
		err = rows.Scan(&deviceID, &user)
		jgh.PanicOnErr(err)

		fmt.Printf("%s is now owned by %s\n", deviceID, user)

		err = setAssetId(deviceID, user, ga)
		jgh.PanicOnErr(err)

		// update admin console table to reflect the change we just made
		_, err = db.Exec(`
			UPDATE chromebooks
			SET asset_id = $1
			WHERE device_id = $2
		`, user, deviceID)
		jgh.PanicOnErr(err)

		// google gets mad if we go too fast
		time.Sleep(time.Second / 2)
	}
	jgh.PanicOnErr(rows.Err())
}

func setAssetId(deviceID, newAssetID string, ga *admin.Service) error {
	_, err := ga.Chromeosdevices.Patch(
		"my_customer",
		deviceID,
		&admin.ChromeOsDevice{
			AnnotatedAssetId: newAssetID,
		},
	).Do()
	return err
}
