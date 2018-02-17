package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

const (
	base      = 36
	baseStr   = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	exDir     = "gen"
	idLimit   = 500
	tryCreate = 3
)

var (
	idRequest  chan ticketReq
	idResponse chan error
)

func createIDs() {
	for {
		_ = <-idRequest
		exists, err := ioutil.ReadDir(exDir)
		if err != nil {
			idResponse <- err
			return
		}
		if len(exists) > idLimit {
			idResponse <- errors.New("ID-limit reached")
			return
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
			idResponse <- errors.New("Couldn't create unique ticket ID")
			return
		}

		// TODO: Create ID on disk (including associated ticket request data)

		idResponse <- nil // TODO: send any relevant ID creation error message
	}

}
