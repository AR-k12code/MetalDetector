package main

import (
	"context"
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

	// get devices in the admin console where the user or asset_id field
	// does not match the value we would assign it
	rows, err := db.Query(
		context.TODO(),
		`
			SELECT
				predictions.device_id,
				COALESCE(predicted_users.first_name || ' ' || predicted_users.last_name || ' <' || predictions.user || '>', 'No recent student logins') AS "user",
				COALESCE(assigned_users.first_name || ' ' || assigned_users.last_name || ' <' || assets.client || '>', 'Not assigned') AS client
			FROM chromebooks
			LEFT JOIN predictions
				ON chromebooks.device_id = predictions.device_id
			LEFT JOIN users AS predicted_users
				ON predictions.user = predicted_users.email
			LEFT JOIN assets
				ON chromebooks.serial = assets.serial
			LEFT JOIN users AS assigned_users
				ON assets.client = assigned_users.email
			WHERE
				chromebooks.user <> COALESCE(predicted_users.first_name || ' ' || predicted_users.last_name || ' <' || predictions.user || '>', 'No recent student logins') OR
				chromebooks.asset_id <> COALESCE(assigned_users.first_name || ' ' || assigned_users.last_name || ' <' || assets.client || '>', 'Not assigned')
		`,
	)
	jgh.PanicOnErr(err)
	defer rows.Close()

	for rows.Next() {
		var deviceID, user, client string
		err = rows.Scan(&deviceID, &user, &client)
		jgh.PanicOnErr(err)

		fmt.Printf("Updateing %s\n", deviceID)

		err = setCustomFields(deviceID, user, client, ga)
		jgh.PanicOnErr(err)

		// update admin console table to reflect the change we just made
		_, err = db.Exec(context.TODO(),
			`
				UPDATE chromebooks
				SET
					"user" = $1,
					asset_id = $2
				WHERE device_id = $3
			`,
			user,
			client,
			deviceID,
		)
		jgh.PanicOnErr(err)

		// google gets mad if we go too fast
		time.Sleep(time.Second / 2)
	}
	jgh.PanicOnErr(rows.Err())
}

func setCustomFields(deviceID, user, asset string, ga *admin.Service) error {
	_, err := ga.Chromeosdevices.Patch(
		"my_customer",
		deviceID,
		&admin.ChromeOsDevice{
			AnnotatedUser:    user,
			AnnotatedAssetId: asset,
		},
	).Do()
	return err
}
