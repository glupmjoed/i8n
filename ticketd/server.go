package main

import (
	"flag"
	"fmt"
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
	port := flag.Arg(0)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		log.Fatal("Please provide a valid port number (1st argument)")
	}

	idRequest = make(chan ticketReq)
	idResponse = make(chan error)
	go createIDs()

	http.HandleFunc(baseURL, http.NotFound)
	http.HandleFunc(baseURL+"order/", orderHandler)

	fmt.Println("Serving ticket requests on port", port, "...")
	http.ListenAndServe(":"+port, nil)
}

func orderHandler(w http.ResponseWriter, r *http.Request) {

	// TODO: Parse and validate form data

	// TODO: Create ticket ID

	// TODO: Display ticket ID and payment options
}
