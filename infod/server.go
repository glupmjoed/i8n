package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func main() {

	flag.Parse()
	port := flag.Arg(0)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		log.Fatal("Please provide a valid port number (1st argument)")

	}

	http.HandleFunc("/", http.NotFound)

	fmt.Println("Serving info requests on port", port, "...")
	http.ListenAndServe(":"+port, nil)
}
