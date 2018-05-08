package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	infoEmailTmpl *template.Template
	infoEmailPath = "info_email.html"
)

func infoHandler(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "info.html")
}

func infoConfHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		msg := fmt.Sprintf("Error parsing HTML form: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, msg)
		return
	}

	b, err := ioutil.ReadFile("config/info_passphrase")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, "Server error: Couldn't read passphrase.")
		return
	}
	if string(bytes.TrimSpace(b)) != r.Form.Get("passphrase") {
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, "Wrong passphrase.")
		return
	}

	id := strings.ToUpper(trunc(r.Form.Get("ticket-id")))
	if !isValidID(id) {
		msg := fmt.Sprintf("Malformed ticket-ID (%s).", id)
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, msg)
		return
	}

	if !ticketExists(id) {
		msg := fmt.Sprintf("Couldn't find ticket with ID %s.", id)
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, msg)
		return
	}

	tck, err := loadID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, "Server error: Couldn't load ticket ID from disk.")
		return
	}

	funcmap := template.FuncMap{
		"add1":   func(n int) int { return n + 1 },
		"single": func() bool { return len(tck.Names) == 1 },
	}
	infoEmailTmpl, err = template.New(infoEmailPath).Funcs(funcmap).
		ParseFiles(infoEmailPath)
	if err != nil {
		msg := fmt.Sprintf("Server error: Template error: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, msg)
		return
	}

	err = infoEmailTmpl.Execute(w, tck)
	if err != nil {
		msg := fmt.Sprintf("Server error: Template error: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, msg)
		return
	}
}

func initInfo() {

	http.HandleFunc(baseURL+"info/", infoHandler)
	http.HandleFunc(baseURL+"info/conf/", infoConfHandler)
}
