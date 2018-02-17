package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
)

type ticketReq struct {
}

const (
	baseURL = "/"
)

func main() {

	flag.Parse()
	if _, err := strconv.ParseUint(flag.Arg(0), 10, 16); err != nil {
		log.Fatal("Please provide a valid port number (1st argument)")
	}

	idRequest = make(chan ticketReq)
	idResponse = make(chan error)
	go createIDs()

	http.HandleFunc(baseURL, http.NotFound)
	http.HandleFunc(baseURL+"order/", orderHandler)
	http.ListenAndServe(":"+flag.Arg(0), nil)
}

func orderHandler(w http.ResponseWriter, r *http.Request) {

	// TODO: Parse and validate form data

	// TODO: Create ticket ID

	// TODO: Display ticket ID and payment options
}
