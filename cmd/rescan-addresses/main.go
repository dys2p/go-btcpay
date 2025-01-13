package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/dys2p/go-btcpay"
)

func main() {

	store, err := btcpay.Load("store.json")
	if err != nil {
		log.Printf("error loading btcpay store: %v", err)
		return
	}

	for {
		fmt.Print(".")
		_, err := store.CreateInvoice(&btcpay.InvoiceRequest{
			Amount:   0,
			Currency: "EUR",
			InvoiceCheckout: btcpay.InvoiceCheckout{
				ExpirationMinutes: 0,
				MonitoringMinutes: 0,
			},
		}) // CreateInvoice(nil) does not work, fails before trying to insert into the database
		if err == nil || !strings.Contains(err.Error(), "500") {
			break
		}
	}

	fmt.Println("\ndone")
}
