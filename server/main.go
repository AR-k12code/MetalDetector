package main

import (
	"context"
	"encoding/json"
	"log"
	"metaldetector/common"
	"net"
	"net/http"
	"strings"

	"github.com/9072997/jgh"
	"github.com/9072997/vas"
	"github.com/foomo/simplecert"
	"github.com/foomo/tlsconfig"
	"github.com/jackc/pgx/v4/pgxpool"
)

// ClientPing represents the data in the JSON report submitted by
// the chromebook every few minutes
type ClientPing struct {
	ChallengeResp string   `json:"challengeResp"`
	Timestamp     *int     `json:"timestamp"`
	SessionStart  *int     `json:"sessionStart"`
	Serial        *string  `json:"serial"`
	LocalIPv4     *string  `json:"ipv4"`
	LocalIPv6     *string  `json:"ipv6"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	Accuracy      *int     `json:"accuracy"`
	GeoTimestamp  *int     `json:"geoTimestamp"`
	Email         *string  `json:"email"`
}

type GetResponse struct {
	Challenge string `json:"challenge"`
}

func httpGetHandler(resp http.ResponseWriter, req *http.Request, db *pgxpool.Pool) {
	// allow CORS if request is from a Chrome extension
	origin := strings.ToLower(req.Header.Get("Origin"))
	if strings.HasPrefix(origin, "chrome-extension://") {
		resp.Header().Set("Access-Control-Allow-Origin", origin)
	}

	resp.Header().Set("Cache-Control", "no-cache")

	// if a redirect URL was specified, use a refresh header to make
	// browsers follow it without interfearing with javascript
	if common.Config.Server.Redirect != "" {
		resp.Header().Set("Refresh", "0;"+common.Config.Server.Redirect)
	}

	vasClient, err := vas.NewFromOAuthClient(common.OAuthClient())
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	challenge, err := vasClient.GetChallenge()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(GetResponse{challenge})
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write(respJSON)
}

func httpPostHandler(resp http.ResponseWriter, req *http.Request, db *pgxpool.Pool) {
	// decode request body json
	var p ClientPing
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	// we wait to verify auth until later for time reasons, but if the
	// client didn't even try, and we are configured to require auth,
	// let them know.
	if common.Config.Server.RequireAuth && p.ChallengeResp == "" {
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(
			"This server requires authentication via the Chrome Verified" +
				"Access API. Please provide a ChallengeResp value in the" +
				"request body\n",
		))
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

	// fork to a goroutine so we can allow the client to hang up ASAP
	go func() {
		// attempt to authenticate the client
		var deviceID string
		if p.ChallengeResp != "" {
			// make an authenticated vas client
			vasClient, err := vas.NewFromOAuthClient(common.OAuthClient())
			if err == nil {
				// use vas client to verify chromebook authentication
				deviceID, err = vasClient.VerifyResponse(p.ChallengeResp, "")
				if err != nil {
					log.Printf("Error authenticating device: %s", err)
				}
			} else {
				log.Printf("VAS error: %s", err)
			}
		}
		// the server can be configured to ignore pings from unauthenticated
		// clients
		if common.Config.Server.RequireAuth && deviceID == "" {
			log.Printf(
				"Authentication failed. Dropping ping from %s: %+v",
				ip, p,
			)
		}

		// to avoid concurency issues with partitions on the database side
		// only run one insert per user or device at a time
		if p.Email == nil {
			Lock("email-")
			defer Unlock("email-")
		} else {
			Lock("email-" + *p.Email)
			defer Unlock("email-" + *p.Email)
		}
		if p.Serial == nil {
			Lock("serial-")
			defer Unlock("serial-")
		} else {
			Lock("serial-" + *p.Serial)
			defer Unlock("serial-" + *p.Serial)
		}

		tx, err := db.Begin(context.TODO())
		jgh.PanicOnErr(err)
		defer tx.Rollback(context.TODO())

		_, err = tx.Exec(context.TODO(),
			`CALL insert_ping($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			ip,
			common.EmptyAsNil(deviceID),
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
			err = tx.Commit(context.TODO())
		}
		if err != nil {
			log.Print(err)
		}
	}()
}

func main() {
	db := common.PGXPool(0)
	defer db.Close()

	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		// check if this is a GET or POST request and divert to the correct
		// handler
		if req.Method == http.MethodGet {
			httpGetHandler(resp, req, db)
		} else {
			httpPostHandler(resp, req, db)
		}
	})
	err := simplecert.ListenAndServeTLSCustom(
		common.Config.Server.Listen,
		nil,
		&common.Config.AutoCert,
		tlsconfig.NewServerTLSConfig(tlsconfig.TLSModeServerStrict),
		nil,
		common.Config.AutoCert.Domains...,
	)
	jgh.PanicOnErr(err)
}
