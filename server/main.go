package main

import (
	"encoding/json"
	"log"
	"metaldetector/common"
	"net"
	"net/http"

	"github.com/9072997/jgh"
	"github.com/foomo/simplecert"
	"github.com/foomo/tlsconfig"
)

// ClientPing represents the data in the JSON report submitted by
// the chromebook every few minutes
type ClientPing struct {
	Timestamp    *int     `json:"timestamp"`
	SessionStart *int     `json:"sessionStart"`
	Serial       *string  `json:"serial"`
	LocalIPv4    *string  `json:"ipv4"`
	LocalIPv6    *string  `json:"ipv6"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	Accuracy     *int     `json:"accuracy"`
	GeoTimestamp *int     `json:"geoTimestamp"`
	Email        *string  `json:"email"`
}

func main() {
	db := common.PGXPool(0)
	defer db.Close()

	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		// check if this is a GET request, then this is not a check-in from
		// the extension. It's probably a user looking for the admin
		// interface. If a redirect URL was provided in the config, send
		// them there.
		isGet := req.Method == http.MethodGet
		if isGet && len(common.Config.Server.Redirect) > 0 {
			http.Redirect(
				resp,
				req,
				common.Config.Server.Redirect,
				http.StatusFound,
			)
			return
		}

		// decode request body json
		var p ClientPing
		err := json.NewDecoder(req.Body).Decode(&p)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}

		// all the response that we send the client is this header
		resp.WriteHeader(http.StatusAccepted)

		// get ip of client
		ip, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		// run prepared statement from above to insert values from client
		// in a goroutine so we can allow the client to hang up ASAP
		go func() {
			tx, err := db.Begin()
			jgh.PanicOnErr(err)
			defer tx.Rollback()

			_, err = tx.Exec(
				`CALL insert_ping($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				ip,
				p.Timestamp,
				p.SessionStart,
				p.Serial,
				p.LocalIPv4,
				p.LocalIPv6,
				p.Latitude,
				p.Longitude,
				p.Accuracy,
				p.GeoTimestamp,
				p.Email,
			)
			if err == nil {
				err = tx.Commit()
			}
			if err != nil {
				log.Print(err)
			}
		}()
	})
	err := simplecert.ListenAndServeTLSCustom(
		common.Config.Server.Listen,
		nil,
		&common.Config.LetsEncrypt,
		tlsconfig.NewServerTLSConfig(tlsconfig.TLSModeServerStrict),
		nil,
		common.Config.LetsEncrypt.Domains...,
	)
	jgh.PanicOnErr(err)
}
