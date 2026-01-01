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

func ParseInvoiceWebhook(t *testing.T) {
	store := &Store{
		Host:          "https://example.org",
		ID:            "my-store-id",
		WebhookSecret: "my-webhook-secret",
		MaxRates:      map[string]float64{"XMR": 1000, "BTC": 500000},
	}

	body := []byte(`
	{
		"deliveryId": "my-delivery-id",
		"webhookId": "my-webhook-id",
		"orignalDeliveryId": "my-original-delivery-id",
		"isRedelivery": false,
		"type": "InvoiceCreated",
		"timestamp": 1610000000,
		"storeId": "my-store-id",
		"invoiceId": "my-invoice-id",
		"metadata": {
			"orderId": "my-order-id"
		}
	}`)

	var webhookRequest = &http.Request{
		Body:   io.NopCloser(bytes.NewReader(body)),
		Header: make(http.Header),
	}
	var mac = hmac.New(sha256.New, []byte(store.WebhookSecret))
	mac.Write(body)
	webhookRequest.Header.Add("BTCPay-Sig", fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil))))

	event, err := store.ParseInvoiceWebhook(webhookRequest)
	if err != nil {
		t.Fatal(err)
	}
	if event.StoreID != store.ID || event.Type != EventInvoiceCreated || event.InvoiceID != "my-invoice-id" {
		t.Fail()
	}
	if event.InvoiceMetadata.OrderID != "my-order-id" {
		t.Fail()
	}
}
