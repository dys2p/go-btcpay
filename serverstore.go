package btcpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrBadRequest      = errors.New("bad request")
	ErrNotFound        = errors.New("not found")
)

type ServerStore struct {
	Host          string             `json:"uri"`        // without "/api" and without trailing slash, used for API access and user links
	HostOnion     string             `json:"onion"`      // without "/api" and without trailing slash, used for user links only, can be empty
	UserAPIKey    string             `json:"userAPIKey"` // to be created in the BTCPay Server user settings (not in the store settings)
	ID            string             `json:"id"`
	WebhookSecret string             `json:"webhookSecret"`
	MaxRates      map[string]float64 `json:"maxRates"` // example: {"XMR": 1000, "BTC": 500000}
}

// Load unmarshals a json config file into a ServerStore.
// If the file doesn't exist, it is created and an error is returned.
func Load(jsonPath string) (*ServerStore, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var store = &ServerStore{}
	return store, json.Unmarshal(data, store)
}

func (s *ServerStore) doRequest(method string, path string, body io.Reader) (*http.Response, error) {

	req, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/api/v1/%s", s.Host, path),
		body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("token %s", s.UserAPIKey))
	req.Header.Add("Content-Type", "application/json")

	return (&http.Client{
		Timeout: 10 * time.Second,
	}).Do(req)
}

// CheckInvoiceAuth checks authentication and authorization by performing bogus CreateInvoice and GetInvoice calls and checking the result.
// It returns ErrUnauthenticated, ErrUnauthorized or nil.
func (s *ServerStore) CheckInvoiceAuth() error {
	if _, err := s.CreateInvoice(nil); err != ErrBadRequest {
		return err
	}
	if _, err := s.GetInvoice("not-existing"); err != ErrNotFound {
		return err
	}
	return nil
}

// CreateInvoice creates an invoice which can be paid by the customer.
// It is recommended to set InvoiceRequest.InvoiceMetadata.OrderID in order to
// identify the order in both a webhook and in your bookkeeping.
// Alternatively you can store the btcpay invoice ID in your order database.
func (s *ServerStore) CreateInvoice(req *InvoiceRequest) (*Invoice, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := s.doRequest(http.MethodPost, fmt.Sprintf("stores/%s/invoices", s.ID), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusUnauthorized: // 401, "Unauthorized" should be "Unauthenticated"
		return nil, ErrUnauthenticated
	case http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusBadRequest:
		return nil, ErrBadRequest
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("response status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var invoice = &Invoice{}
	return invoice, json.Unmarshal(body, invoice)
}

func (s *ServerStore) GetInvoice(id string) (*Invoice, error) {

	resp, err := s.doRequest(http.MethodGet, fmt.Sprintf("stores/%s/invoices/%s", s.ID, id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusUnauthorized: // 401, "Unauthorized" should be "Unauthenticated"
		return nil, ErrUnauthenticated
	case http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusBadRequest:
		return nil, ErrBadRequest
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("response status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var invoice = &Invoice{}
	return invoice, json.Unmarshal(body, invoice)
}

func (s *ServerStore) GetInvoicePaymentMethods(id string) ([]InvoicePaymentMethod, error) {

	resp, err := s.doRequest(http.MethodGet, fmt.Sprintf("stores/%s/invoices/%s/payment-methods", s.ID, id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusForbidden: // 403
		return nil, ErrUnauthorized
	case http.StatusNotFound: // 404
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("response status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var invoice = []InvoicePaymentMethod{}
	return invoice, json.Unmarshal(body, &invoice)
}

// GetServerStatus requires successful authentication, but no specific permissions.
func (s *ServerStore) GetServerStatus() (*ServerStatus, error) {

	resp, err := s.doRequest(http.MethodGet, "server/info", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusUnauthorized: // 401, "Unauthorized" should be "Unauthenticated"
		return nil, ErrUnauthenticated
	case http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusBadRequest:
		return nil, ErrBadRequest
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("response status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var status = &ServerStatus{}
	return status, json.Unmarshal(body, status)
}

func (s *ServerStore) InvoiceCheckoutLink(id string) string {
	return fmt.Sprintf("%s/i/%s", s.Host, id)
}

func (s *ServerStore) InvoiceCheckoutLinkPreferOnion(id string) string {
	host := s.Host
	if s.HostOnion != "" {
		host = s.HostOnion
	}
	return fmt.Sprintf("%s/i/%s", host, id)
}

func (s *ServerStore) ProcessWebhook(r *http.Request) (*InvoiceEvent, error) {

	var messageMAC = []byte(strings.TrimPrefix(r.Header.Get("BTCPay-Sig"), "sha256="))
	if len(messageMAC) == 0 {
		return nil, errors.New("BTCPay-Sig header missing")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	var mac = hmac.New(sha256.New, []byte(s.WebhookSecret))
	mac.Write(body)
	var expectedMAC = []byte(hex.EncodeToString(mac.Sum(nil)))
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, fmt.Errorf("HMAC mismatch, got %s, want %s", messageMAC, expectedMAC)
	}

	var event = &InvoiceEvent{}
	if err := json.Unmarshal(body, event); err != nil {
		return nil, fmt.Errorf("unmarshaling body: %w", err)
	}

	// mitigate BTCPayServer misconfigurations by checking the store ID
	if event.StoreID != s.ID {
		return nil, fmt.Errorf("invoice store ID %s does not match selected store ID %s", event.StoreID, s.ID)
	}

	// mitigate invalid rates
	paymentMethods, err := s.GetInvoicePaymentMethods(event.InvoiceID)
	if err != nil {
		return nil, fmt.Errorf("getting payment methods from invoice: %w", err)
	}
	for cryptoCode, maxRate := range s.MaxRates {
		if err := ValidateRate(paymentMethods, cryptoCode, maxRate); err != nil {
			return nil, fmt.Errorf("validating rate: %w", err)
		}
	}

	return event, nil
}
