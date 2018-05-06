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
	failTmpl  *template.Template
	orderTmpl *template.Template
	payTmpl   *template.Template
)

func main() {

	flag.Parse()
	port := flag.Arg(0)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		log.Fatal("Please provide a valid port number (1st argument)")
	}

	failTmpl = template.Must(template.ParseFiles(tmplDir + "fail.html"))
	orderTmpl = template.Must(template.ParseFiles(tmplDir + "order.html"))
	payTmpl = template.Must(template.ParseFiles(tmplDir + "pay.html"))

	idRequest = make(chan *ticketReq)
	idResponse = make(chan error)
	go handleIDRequests()

	b, err := ioutil.ReadFile("config/stripe_secret.key")
	if err != nil {
		log.Fatal("Couldn't read Stripe key: " + err.Error())
	}
	stripe.Key = string(bytes.TrimSpace(b))

	http.HandleFunc(baseURL, http.NotFound)
	http.HandleFunc(baseURL+"order/", orderHandler)
	http.HandleFunc(baseURL+"pay/", payHandler)

	initInfo()

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
		msg := fmt.Sprintf("Enter at least one name and an amount >= %d DKK",
			minAmount)
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
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, msg)
		return
	}
	tck := ticketReq{
		Time:  time.Now().UTC().Format(time.UnixDate),
		Email: template.HTMLEscapeString(trunc(r.Form.Get("email"))),
	}

	tck.Names, tck.Amounts, err = getNameAmountPairs(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: Sanitize error message
		failTmpl.Execute(w, err.Error())
		return
	}

	for _, amount := range tck.Amounts {
		tck.AmountTotal += amount
	}
	if tck.AmountTotal > maxAmount {
		msg := fmt.Sprintf("Please enter a combined ticket price <= %d DKK",
			maxAmount)
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, msg)
		return
	}

	if tck.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, "Please provide an email address")
		return
	}

	err = createID(&tck)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Sanitize error message
		failTmpl.Execute(w, err.Error())
		return
	}

	orderTmpl.Execute(w, tck)
}

func payHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		msg := fmt.Sprintf("Couldn't parse HTML form: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, msg)
		return
	}

	token := template.HTMLEscapeString(r.Form.Get("stripeToken"))
	if token == "" {
		msg := "Missing Stripe token. Make sure that JavaScript is enabled"
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, msg)
		return
	}

	ticketID := template.HTMLEscapeString(r.Form.Get("ticket-id"))
	if ticketID == "" {
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, "Missing ticket ID")
		return
	}

	if ticketSaved(ticketID) {
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, "Ticket is already paid for")
		return
	}

	var tck *ticketReq
	tck, err = loadID(ticketID)
	if err != nil {
		// TODO: Sanitize error message
		msg := fmt.Sprintf("Couldn't find ticket ID %s: %s",
			ticketID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		failTmpl.Execute(w, msg)
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

		// TODO: Sanitize error message
		var errStr string
		if stripeErr, ok := err.(*stripe.Error); ok {
			switch stripeErr.Code {
			case stripe.IncorrectNum:
				errStr = "IncorrectNum"
			case stripe.InvalidNum:
				errStr = "InvalidNum"
			case stripe.InvalidExpM:
				errStr = "InvalidExpM"
			case stripe.InvalidExpY:
				errStr = "InvalidExpY"
			case stripe.InvalidCvc:
				errStr = "InvalidCvc"
			case stripe.ExpiredCard:
				errStr = "ExpiredCard"
			case stripe.IncorrectCvc:
				errStr = "IncorrectCvc"
			case stripe.CardDeclined:
				errStr = "CardDeclined"
			case stripe.Missing:
				errStr = "Missing"
			case stripe.ProcessingErr:
				errStr = "ProcessingErr"
			default:
				errStr = stripeErr.Error()
			}
		} else {
			errStr = chargeErr.Error()
		}
		msg := fmt.Sprintf("Payment failed (charge error): %s\n",
			errStr)
		fmt.Fprintf(os.Stderr, msg)
		// TODO: Base HTTP header on error type
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, msg)
		return
	}

	if !charge.Paid {
		msg := fmt.Sprintf("Payment failed: %s: %s\n",
			charge.FailCode, charge.FailMsg)
		fmt.Fprintf(os.Stderr, msg)
		w.WriteHeader(http.StatusBadRequest)
		failTmpl.Execute(w, msg)
		return
	}

	if charge.Currency != "dkk" || charge.Amount != tck.AmountTotal*100 {
		// TODO: Handle non-conforming charges
	}

	err = saveTicket(tck)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't save ticket: %s\n", err.Error())
	}

	payTmpl.Execute(w, tck)
}

func trunc(s string) string {
	return fmt.Sprintf("%.100s", s)
}
