package main

import (
	"net/http"
)

func infoHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
