package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/9072997/jgh"
	"github.com/foomo/simplecert"
	"github.com/foomo/tlsconfig"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/copier"
	"github.com/pelletier/go-toml"
)

// Config contains values from the config file
var Config = struct {
	MySQL       mysql.Config
	LetsEncrypt simplecert.Config
	Listen      string
}{}

// MarshalableConfig contains all the fields from the config we can
// serialize to TOML
type MarshalableConfig struct {
	Listen      string
	LetsEncrypt struct {
		RenewBefore   int
		CheckInterval time.Duration
		SSLEmail      string
		DirectoryURL  string
		HTTPAddress   string
		TLSAddress    string
		CacheDirPerm  os.FileMode
		Domains       []string
		DNSServers    []string
		CacheDir      string
		DNSProvider   string
		Local         bool
		UpdateHosts   bool
	}
	MySQL struct {
		User                    string
		Passwd                  string
		Net                     string
		Addr                    string
		DBName                  string
		Collation               string
		MaxAllowedPacket        int
		ServerPubKey            string
		TLSConfig               string
		Timeout                 time.Duration
		ReadTimeout             time.Duration
		WriteTimeout            time.Duration
		AllowAllFiles           bool
		AllowCleartextPasswords bool
		AllowNativePasswords    bool
		AllowOldPasswords       bool
		CheckConnLiveness       bool
		ClientFoundRows         bool
		ColumnsWithAlias        bool
		InterpolateParams       bool
		MultiStatements         bool
		ParseTime               bool
		RejectReadOnly          bool
		Params                  map[string]string
	}
}

func init() {
	// set defaults
	copier.Copy(&Config.MySQL, mysql.NewConfig())
	copier.Copy(&Config.LetsEncrypt, simplecert.Default)
	Config.LetsEncrypt.TLSAddress = "" // we probably use port 443
	Config.Listen = ":443"

	// get "config.toml" file next to this executable
	exePath, err := os.Executable()
	jgh.PanicOnErr(err)
	ConfigDir := filepath.Dir(exePath)
	configFile := filepath.Join(ConfigDir, "config.toml")
	configTOML, err := ioutil.ReadFile(configFile)
	jgh.PanicOnErr(err)

	// read config file into global Config variable
	err = toml.Unmarshal(configTOML, &Config)
	jgh.PanicOnErr(err)

	// print config with defaults filled in for debugging/ease of setup
	var marshalableConfig MarshalableConfig
	copier.Copy(&marshalableConfig, Config)
	toml.
		NewEncoder(os.Stdout).
		Order(toml.OrderPreserve).
		Indentation("").
		Encode(marshalableConfig)
	fmt.Println()
}

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
	db, err := sql.Open("mysql", Config.MySQL.FormatDSN())
	jgh.PanicOnErr(err)

	err = db.Ping()
	jgh.PanicOnErr(err)

	stmt, err := db.Prepare(`
		INSERT DELAYED INTO pings (
			requestIP,
			clientTime,
			sessionStart,
			serial,
			localIPv4,
			localIPv6,
			location,
			accuracy,
			geoTime,
			email
		) VALUES (
			?,
			FROM_UNIXTIME(?),
			FROM_UNIXTIME(?),
			?,
			?,
			?,
			POINT(?, ?),
			?,
			FROM_UNIXTIME(?),
			?
		)
	`)
	jgh.PanicOnErr(err)

	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
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
			_, err := stmt.Exec(
				ip,
				p.Timestamp,
				p.SessionStart,
				p.Serial,
				p.LocalIPv4,
				p.LocalIPv6,
				p.Longitude, p.Latitude,
				p.Accuracy,
				p.GeoTimestamp,
				p.Email,
			)
			if err != nil {
				log.Print(err)
			}
		}()
	})
	err = simplecert.ListenAndServeTLSCustom(
		Config.Listen,
		nil,
		&Config.LetsEncrypt,
		tlsconfig.NewServerTLSConfig(tlsconfig.TLSModeServerStrict),
		nil,
		Config.LetsEncrypt.Domains...,
	)
	jgh.PanicOnErr(err)
}
