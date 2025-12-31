package btcpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestInvoice(t *testing.T) {

	store, err := Load("store.json")
	if err != nil {
		t.Fatal(err)
	}

	ir := &InvoiceRequest{
		Amount:   1.23,
		Currency: "EUR",
	}
	ir.OrderID = "Test"

	created, err := store.CreateInvoice(ir)
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetInvoice(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Amount != 1.23 || got.Currency != "EUR" {
		t.Fail()
	}

	// simulate a webhook

	var body = []byte(fmt.Sprintf(`
	{
		"deliveryId": "delivery-abcdefghijkl",
		"webhookId": "webhook-abcdefghijklmn",
		"orignalDeliveryId": "delivery-abcdefghijkl",
		"isRedelivery": false,
		"type": "InvoiceCreated",
		"timestamp": 1610000000,
		"storeId": "%s",
		"invoiceId": "%s",
		"metadata": {
			"orderId": "%s"
		}
	}`, store.ID, got.ID, got.InvoiceMetadata.OrderID))

	var webhookRequest = &http.Request{
		Body:   io.NopCloser(bytes.NewReader(body)),
		Header: make(http.Header),
	}
	var mac = hmac.New(sha256.New, []byte(store.WebhookSecret))
	mac.Write(body)
	webhookRequest.Header.Add("BTCPay-Sig", fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil))))

	event, err := store.ProcessWebhook(webhookRequest)
	if err != nil {
		t.Fatal(err)
	}

	if event.StoreID != store.ID || event.Type != EventInvoiceCreated || event.InvoiceID != got.ID {
		t.Fail()
	}

	if event.InvoiceMetadata.OrderID != got.InvoiceMetadata.OrderID {
		t.Fail()
	}
}
