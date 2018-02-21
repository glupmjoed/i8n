package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
)

type ticketReq struct {
	ID          string
	Time        string
	Email       string
	Names       []string
	Amounts     []uint64
	AmountTotal uint64
}

const (
	minAmount = 5
	maxAmount = 10000
	baseURL   = "/"
	tmplDir   = "../"
)

var (
	orderTmpl       *template.Template
	payTmpl         *template.Template
	payFailTmpl     *template.Template
	stripeSecretKey string
)

func main() {

	flag.Parse()
	port := flag.Arg(0)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		log.Fatal("Please provide a valid port number (1st argument)")
	}

	orderTmpl = template.Must(template.ParseFiles(tmplDir + "order.html"))
	payTmpl = template.Must(template.ParseFiles(tmplDir + "pay.html"))
	payFailTmpl = template.Must(template.ParseFiles(tmplDir + "pay_fail.html"))

	idRequest = make(chan *ticketReq)
	idResponse = make(chan error)
	go handleIDRequests()

	b, err := ioutil.ReadFile("config/stripe_secret.key")
	if err != nil {
		log.Fatal("couldn't read Stripe key: " + err.Error())
	}
	stripeSecretKey = string(bytes.TrimSpace(b))

	http.HandleFunc(baseURL, http.NotFound)
	http.HandleFunc(baseURL+"order/", orderHandler)
	http.HandleFunc(baseURL+"order/pay/", payHandler)

	fmt.Println("Serving ticket requests on port", port, "...")
	http.ListenAndServe(":"+port, nil)
}

func getNameAmountPair(f url.Values, n int) (string, uint64) {
	name := template.HTMLEscapeString(trunc(f.Get(fmt.Sprintf("name%d", n))))
	amount, err := strconv.ParseUint(f.Get(fmt.Sprintf("amount%d", n)), 10, 64)
	if err != nil || name == "" || amount < minAmount {
		return "", 0
	}
	return name, amount
}

func getNameAmountPairs(f url.Values) ([]string, []uint64, error) {
	var names []string
	var amounts []uint64
	name, amount := getNameAmountPair(f, 1)
	if name == "" {
		msg := "Enter at least one name and an amount >= 5 DKK"
		return nil, nil, errors.New(msg)
	}
	for i := 2; name != ""; i++ {
		names = append(names, name)
		amounts = append(amounts, amount)
		name, amount = getNameAmountPair(f, i)
	}
	return names, amounts, nil
}

func orderHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		msg := fmt.Sprintf("Error parsing HTML form: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	tck := ticketReq{
		Time:  time.Now().UTC().Format(time.UnixDate),
		Email: template.HTMLEscapeString(trunc(r.Form.Get("email"))),
	}

	tck.Names, tck.Amounts, err = getNameAmountPairs(r.Form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		// TODO: Return prettier error message
		return
	}

	for _, amount := range tck.Amounts {
		tck.AmountTotal += amount
		if amount > maxAmount {
			msg := fmt.Sprintf("Please enter a price <= %d DKK", maxAmount)
			http.Error(w, msg, http.StatusBadRequest)
			// TODO: Return prettier error message
		}
	}

	if tck.Email == "" {
		http.Error(w, "Please provide an email address", http.StatusBadRequest)
		// TODO: Return prettier error message
		return
	}

	err = createID(&tck)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// TODO: Return prettier error message
		return
	}

	orderTmpl.Execute(w, tck)
}

func payHandler(w http.ResponseWriter, r *http.Request) {

	stripe.Key = stripeSecretKey
	// TODO: Move key initialization to main function

	err := r.ParseForm()
	if err != nil {
		msg := fmt.Sprintf("Error parsing HTML form: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		// TODO: Return prettier error message
		return
	}

	token := template.HTMLEscapeString(r.Form.Get("stripeToken"))
	if token == "" {
		http.Error(w, "Missing Stripe token. Is JavaScript enabled?",
			http.StatusBadRequest)
		// TODO: Return prettier error message
		return
	}

	ticketID := template.HTMLEscapeString(r.Form.Get("ticket-id"))
	if ticketID == "" {
		http.Error(w, "Missing ticket ID", http.StatusBadRequest)
		// TODO: Return prettier error message
		return
	}

	if ticketSaved(ticketID) {
		http.Error(w, "Ticket is already paid for", http.StatusBadRequest)
		// TODO: Return prettier error message
		return
	}

	var tck *ticketReq
	tck, err = loadID(ticketID)
	if err != nil {
		msg := fmt.Sprintf("Couldn't find ticket ID %s: %s",
			ticketID, err.Error())
		http.Error(w, msg, http.StatusInternalServerError)
	}

	params := &stripe.ChargeParams{
		Amount:    tck.AmountTotal * 100,
		Currency:  "dkk",
		Desc:      tck.ID,
		Email:     tck.Email,
		Statement: "Ild i Gilden " + tck.ID,
	}
	params.SetSource(token)

	charge, chargeErr := charge.New(params)
	if chargeErr != nil {
		msg := "non-nil charge error: " + chargeErr.Error()
		fmt.Fprintf(os.Stderr, msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if !charge.Paid {
		msg := "Payment failed: " + charge.FailCode + charge.FailMsg
		fmt.Fprintf(os.Stderr, msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if charge.Currency != "dkk" || charge.Amount != tck.AmountTotal*100 {
		// TODO: Handle non-conforming charges
	}

	err = saveTicket(tck)
	if err != nil {
		msg := "Couldn't save ticket: " + err.Error()
		fmt.Fprintf(os.Stderr, msg)
	}

	payTmpl.Execute(w, tck)
}

func trunc(s string) string {
	return fmt.Sprintf("%.100s", s)
}
