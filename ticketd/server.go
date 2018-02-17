package main

type ticketReq struct {
}

func main() {
	idRequest = make(chan ticketReq)
	idResponse = make(chan error)
}
