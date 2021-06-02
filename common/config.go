package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/9072997/jgh"
	"github.com/foomo/simplecert"
	"github.com/jackc/pgx/v4"
	"github.com/jinzhu/copier"
	toml "github.com/pelletier/go-toml"
)

// Config contains values from the main config file
var Config = struct {
	Server struct {
		Listen      string
		Redirect    string
		RequireAuth bool
	}
	AutoCert simplecert.Config
	PgSQL    pgx.ConnPoolConfig
	ESchool  struct {
		APIKey    string
		ReportURL string
	}
	LDAP struct {
		URL        string
		UserDN     string
		Password   string
		BaseDN     string
		Filter     string
		DateFormat string
		PageSize   int
	}
	Helpdesk struct {
		BaseURL              string
		APIKey               string
		User                 string
		MaxConcurentRequests int
	}
}{}

// MarshalableConfig contains all the fields from the config we can
// serialize to TOML
type MarshalableConfig struct {
	Server struct {
		Listen      string
		Redirect    string
		RequireAuth bool
	}
	AutoCert struct {
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
	PgSQL struct {
		Host                 string
		Port                 uint16
		Database             string
		User                 string
		Password             string
		MaxConnections       int
		AcquireTimeout       time.Duration
		PreferSimpleProtocol bool
		RuntimeParams        map[string]string
	}
	ESchool struct {
		APIKey    string
		ReportURL string
	}
	LDAP struct {
		URL        string
		UserDN     string
		Password   string
		BaseDN     string
		Filter     string
		DateFormat string
		PageSize   int
	}
	Helpdesk struct {
		BaseURL              string
		APIKey               string
		User                 string
		MaxConcurentRequests int
	}
}

// read in the config
func init() {
	// set defaults
	copier.Copy(&Config.AutoCert, simplecert.Default)
	Config.AutoCert.TLSAddress = "" // we probably use port 443 elsewhere
	Config.Server.Listen = ":443"
	Config.LDAP.Filter = "(&(objectClass=user)(objectCategory=Person))"
	Config.LDAP.DateFormat = "20060102150405.0Z07"
	Config.LDAP.PageSize = 10
	Config.Helpdesk.MaxConcurentRequests = 1

	// get "config.toml" file next to this executable
	configTOML, err := ioutil.ReadFile(Path("config.toml"))
	if os.IsNotExist(err) {
		fmt.Println("WARNING: no config.toml found. Using defaults.")
	} else {
		jgh.PanicOnErr(err)
	}

	// read config file into global Config variable
	err = toml.Unmarshal(configTOML, &Config)
	jgh.PanicOnErr(err)

	// print config with defaults filled in for debugging/ease of setup
	var marshalableConfig MarshalableConfig
	copier.Copy(&marshalableConfig, Config)
	// hide passwords & keys
	hide(&marshalableConfig.PgSQL.Password)
	hide(&marshalableConfig.ESchool.APIKey)
	hide(&marshalableConfig.LDAP.Password)
	hide(&marshalableConfig.Helpdesk.APIKey)
	// print as TOML
	toml.
		NewEncoder(os.Stdout).
		Order(toml.OrderPreserve).
		Indentation("").
		Encode(marshalableConfig)
	fmt.Println()
}

var workingDirectoryMode = false

// get the full path to a config file next to this exe
func Path(filename string) string {
	// get path to file next to this EXE
	exePath, err := os.Executable()
	jgh.PanicOnErr(err)
	configDir := filepath.Dir(exePath)
	path := filepath.Join(configDir, filename)

	// if we have previously been fetching files from the working directory
	// continue doing that
	if workingDirectoryMode {
		return filename
	}

	// if the file does not exist, but does exist in the current working
	// directory, return the path to the one in the working directory
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// check if it exists in the working directory
		_, err = os.Stat(filename)
		if err == nil {
			workingDirectoryMode = true
			return filename
		}
	}

	return path
}

// replace a string wih the appropriate number of *s
func hide(s *string) {
	*s = strings.Repeat("*", len(*s))
}
