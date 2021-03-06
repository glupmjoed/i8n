package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	base      = 36
	baseStr   = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	exDir     = "ex"
	idLen     = 8
	idLimit   = 1000
	idPrefix  = "IG18"
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
	buf, err := json.MarshalIndent(tck, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	return ioutil.WriteFile(saveDir+"/"+tck.ID, buf, 0640)
}

func ticketExists(id string) bool {
	_, err := os.Stat(exDir + "/" + id)
	return err == nil
}

func ticketSaved(id string) bool {
	_, err := os.Stat(saveDir + "/" + id)
	return err == nil
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
		newID = fmt.Sprintf(idPrefix+"%c%c%c%c",
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

func isValidID(id string) bool {
	if len(id) != idLen || !strings.HasPrefix(id, idPrefix) {
		return false
	}
	for _, r := range id[len(idPrefix):] {
		if (r < '0' || '9' < r) && (r < 'A' || 'Z' < r) {
			return false
		}
	}
	return true
}
