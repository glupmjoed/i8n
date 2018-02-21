package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

const (
	base      = 36
	baseStr   = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	exDir     = "gen"
	idLimit   = 1000
	saveDir   = "save"
	tryCreate = 3
)

var (
	idRequest  chan *ticketReq
	idResponse chan error
)

func createID(r *ticketReq) error {
	idRequest <- r
	return <-idResponse
}

func handleIDRequests() {
	for {
		idResponse <- unsafeCreateID(<-idRequest)
	}
}

func loadID(id string) (*ticketReq, error) {
	buf, err := ioutil.ReadFile(exDir + "/" + id)
	if err != nil {
		return nil, err
	}
	var tck ticketReq
	err = json.Unmarshal(buf, &tck)
	if err != nil {
		return nil, err
	}
	return &tck, nil
}

func saveTicket(tck *ticketReq) error {
	buf, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	return ioutil.WriteFile(saveDir+"/"+newID, buf, 0640)
}

func unsafeCreateID(r *ticketReq) error {
	exists, err := ioutil.ReadDir(exDir)
	if err != nil {
		return err
	}
	if len(exists) > idLimit {
		return errors.New("ID-limit reached")
	}
	enumPos := (base*base - len(exists) - 1) % (base * base)
	var newID string
	idExists := false
	for try := 0; try < tryCreate; try++ {
		timePrt := time.Now().UnixNano() % (base * base)
		newID = fmt.Sprintf("IG18%c%c%c%c",
			baseStr[timePrt/base], baseStr[timePrt%base],
			baseStr[enumPos/base], baseStr[enumPos%base])

		for _, id := range exists {
			if newID == id.Name() {
				idExists = true
				break
			}
		}
		if idExists {
			continue
		}
		break
	}
	if idExists {
		return errors.New("Couldn't create unique ticket ID")
	}

	r.ID = newID
	var buf []byte
	buf, err = json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	return ioutil.WriteFile(exDir+"/"+newID, buf, 0640)
}
