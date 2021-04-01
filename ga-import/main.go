package main

import (
	"context"
	"fmt"
	"metaldetector/common"
	"time"

	"github.com/9072997/jgh"
	"github.com/jackc/pgx"
	admin "google.golang.org/api/admin/directory/v1"
)

func main() {
	// it's a single transaction, so only allow a single connection
	db := common.PGXPool(1)
	defer db.Close()

	// get authorized google admin client (may prompt for oauth flow)
	ga := common.GAdmin()

	jgh.Try(120, 2, true, "", func() bool {
		// start a transaction. If anything goes wrong, rollback.
		tx, err := db.Begin()
		jgh.PanicOnErr(err)
		defer tx.Rollback()

		// clear the table
		_, err = tx.Exec("TRUNCATE chromebooks")
		jgh.PanicOnErr(err)

		deviceCount := 0
		err = ga.Chromeosdevices.List("my_customer").Pages(
			context.Background(),
			func(page *admin.ChromeOsDevices) error {
				for _, device := range page.Chromeosdevices {
					err = device2db(*device, tx)
					jgh.PanicOnErr(err)
					deviceCount++
				}
				fmt.Printf("Devices: %d\n", deviceCount)
				// google gets mad if we go too fast
				time.Sleep(time.Second / 2)
				return nil
			},
		)
		jgh.PanicOnErr(err)

		// commit transaction
		err = tx.Commit()
		jgh.PanicOnErr(err)

		return true
	})
}

func device2db(device admin.ChromeOsDevice, tx *pgx.Tx) error {
	var lanIP, wanIP string
	if len(device.LastKnownNetwork) >= 1 {
		lanIP = device.LastKnownNetwork[0].IpAddress
		wanIP = device.LastKnownNetwork[0].WanIpAddress
	}

	var recentUsers []string
	for _, user := range device.RecentUsers {
		recentUsers = append(recentUsers, user.Email)
	}

	var isDevMode *bool
	switch device.BootMode {
	case "Dev":
		isDevMode = new(bool)
		*isDevMode = true
	case "Verified":
		isDevMode = new(bool)
		*isDevMode = false
	}

	_, err := tx.Exec(
		`
			INSERT INTO chromebooks (
				serial,
				device_id,
				status,
				last_sync,
				"user",
				location,
				asset_id,
				notes,
				model,
				os_version,
				wifi_mac,
				ethernet_mac,
				dev_mode,
				enrollment_time,
				org_unit,
				recent_users,
				lan_ip,
				wan_ip
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
				$11,$12,$13,$14,$15,$16,$17,$18
			)
		`,
		common.EmptyAsNil(device.SerialNumber),
		common.EmptyAsNil(device.DeviceId),
		common.EmptyAsNil(device.Status),
		common.EmptyAsNil(device.LastSync),
		common.EmptyAsNil(device.AnnotatedUser),
		common.EmptyAsNil(device.AnnotatedLocation),
		common.EmptyAsNil(device.AnnotatedAssetId),
		common.EmptyAsNil(device.Notes),
		common.EmptyAsNil(device.Model),
		common.EmptyAsNil(device.OsVersion),
		common.EmptyAsNil(device.MacAddress),
		common.EmptyAsNil(device.EthernetMacAddress),
		isDevMode,
		common.EmptyAsNil(device.LastEnrollmentTime),
		common.EmptyAsNil(device.OrgUnitPath),
		recentUsers,
		common.EmptyAsNil(lanIP),
		common.EmptyAsNil(wanIP),
	)
	return err
}
