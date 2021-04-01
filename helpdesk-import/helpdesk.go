package main

import (
	"fmt"
	"log"
	"metaldetector/common"
	"net/url"
	"strconv"

	"github.com/9072997/jgh"
)

type Status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Client struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Notes     string `json:"notes"`
	Username  string `json:"username"`
	Location  struct {
		LocationName string `json:"locationName"`
	} `json:"location"`
}

type Asset struct {
	ID           int      `json:"id"`
	AssetNumber  string   `json:"assetNumber"`
	AssetStatus  Status   `json:"assetstatus"`
	Notes        string   `json:"notes"`
	SerialNumber string   `json:"serialNumber"`
	Clients      []Client `json:"clients"`
	Room         struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		RoomName string `json:"roomName"`
	} `json:"room"`
	Location struct {
		Name string `json:"locationName"`
	} `json:"location"`
	Model struct {
		Name string `json:"modelName"`
	} `json:"model"`
}

func hreq(
	method string,
	path string,
	params map[string]string,
	reqObj map[string]interface{},
	respObj interface{},
) error {
	// build URL with API key and all params
	u := common.Config.Helpdesk.BaseURL + path +
		"?apiKey=" + url.QueryEscape(common.Config.Helpdesk.APIKey)
	if common.Config.Helpdesk.User != "" {
		u += "&username=" + url.QueryEscape(common.Config.Helpdesk.User)
	}
	for key, value := range params {
		u += "&" + url.QueryEscape(key) + "=" + url.QueryEscape(value)
	}

	ok, msg := jgh.Try(1, 3, false, "", func() bool {
		status, _ := jgh.RESTRequest(
			nil,
			method,
			u,
			"",
			"",
			nil,
			reqObj,
			&respObj,
		)
		if status < 200 || status >= 300 {
			panic("Expected 2xx HTTP status")
		}
		return true
	})
	if !ok {
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func asset(serialNumber string) (Asset, error) {
	// do a search by serial number to find the asset ID
	var results []Asset
	err := hreq(
		"GET",
		"/Assets",
		map[string]string{
			"qualifier": "((serialNumber caseInsensitiveLike '" + serialNumber + "') AND (deleted=null OR deleted=0))",
		},
		nil,
		&results,
	)
	if err != nil {
		return Asset{}, err
	}
	if len(results) == 0 {
		return Asset{},
			fmt.Errorf("serial number not found: %s", serialNumber)
	}
	if len(results) > 1 {
		return Asset{},
			fmt.Errorf("multiple assets with serial number: %s", serialNumber)
	}
	assetID := results[0].ID

	// fetch asset by ID
	var asset Asset
	err = hreq(
		"GET",
		"/Assets/"+strconv.Itoa(assetID),
		nil,
		nil,
		&asset,
	)
	if err != nil {
		return Asset{}, err
	}

	// map numeric IDs status name
	asset.AssetStatus.Name = statusName(asset.AssetStatus.ID)

	// add full clients in place of ID-only stubs
	for i, c := range asset.Clients {
		asset.Clients[i], err = client(c.ID)
		if err != nil {
			log.Println("WARNING: Error finding client with ID", c.ID)
		}
	}

	return asset, nil
}

func client(id int) (c Client, err error) {
	err = hreq(
		"GET",
		"/Clients/"+strconv.Itoa(id),
		nil,
		nil,
		&c,
	)
	return
}

// cache of asset status id -> name
var statuses map[int]string

// map status ID to status name using a cache
func statusName(id int) string {
	if statuses == nil {
		// get all statuses
		var resp []Status
		err := hreq(
			"GET",
			"/AssetStatuses",
			map[string]string{
				// this is the max limit we can set
				// without changeing settings
				"limit": "1000",
			},
			nil,
			&resp,
		)
		jgh.PanicOnErr(err)

		// save them as a map of id -> name
		statuses = make(map[int]string)
		for _, status := range resp {
			statuses[status.ID] = status.Name
		}
	}

	return statuses[id]
}
