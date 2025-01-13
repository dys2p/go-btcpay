package btcpay

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// DummyStore is designed for testing only. It supports invoices only and is not thread-safe.
type DummyStore struct {
	Invoices map[string]*Invoice
}

func NewDummyStore() *DummyStore {
	return &DummyStore{
		Invoices: make(map[string]*Invoice),
	}
}

func (*DummyStore) CheckInvoiceAuth() error {
	return nil
}

func (s *DummyStore) CreateInvoice(req *InvoiceRequest) (*Invoice, error) {
	id := fmt.Sprintf("dummy-invoice-%d", time.Now().UnixNano())
	invoice := &Invoice{
		InvoiceRequest:       *req,
		ID:                   id,
		CheckoutLink:         "http://example.com",
		CreatedTime:          time.Now().Unix(),
		ExpirationTime:       time.Now().Unix() + int64(60*req.ExpirationMinutes),
		MonitoringExpiration: time.Now().Unix() + int64(60*req.MonitoringMinutes),
		Status:               InvoiceNew,
	}
	s.Invoices[id] = invoice
	return invoice, nil
}

func (*DummyStore) CreatePaymentRequest(req *PaymentRequestRequest) (*PaymentRequest, error) {
	return nil, errors.New("not implemented")
}

func (s *DummyStore) GetInvoice(id string) (*Invoice, error) {
	invoice, ok := s.Invoices[id]
	if ok {
		return invoice, nil
	} else {
		return nil, errors.New("not found")
	}
}

func (*DummyStore) GetPaymentRequest(id string) (*PaymentRequest, error) {
	return nil, errors.New("not implemented")
}

func (*DummyStore) GetServerStatus() (*ServerStatus, error) {
	return &ServerStatus{
		Version:                 "dummy",
		SupportedPaymentMethods: []string{"BTC-CHAIN"},
		FullySynched:            true,
		SyncStatuses: []SyncStatus{
			{
				PaymentMethodID: "BTC-CHAIN",
				ChainHeight:     1000000,
				SyncHeight:      1000000,
				Available:       true,
			},
		},
	}, nil
}

func (*DummyStore) InvoiceCheckoutLink(id string) string {
	return id
}

func (*DummyStore) InvoiceCheckoutLinkPreferOnion(id string) string {
	return id
}

func (*DummyStore) PaymentRequestLink(id string) string {
	return id
}

func (*DummyStore) PaymentRequestLinkPreferOnion(id string) string {
	return id
}

func (*DummyStore) ProcessWebhook(r *http.Request) (*InvoiceEvent, error) {
	return nil, errors.New("not implemented")
}
