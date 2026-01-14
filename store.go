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
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	ErrBadRequest      = errors.New("bad request")
	ErrNotFound        = errors.New("not found")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrWebhookSig      = errors.New("BTCPay-Sig header missing or HMAC mismatch") // may be triggered without authentication
)

type ServerStatus struct {
	SyncStatuses            []SyncStatus `json:"syncStatus"`
	Version                 string       `json:"version"`
	Onion                   string       `json:"onion"`
	SupportedPaymentMethods []string     `json:"supportedPaymentMethods"`
	FullySynched            bool         `json:"fullySynched"`
}

type SyncStatus struct {
	ChainHeight     int `json:"chainHeight,omitempty"`
	SyncHeight      int `json:"syncHeight,omitempty"`
	NodeInformation struct {
		Headers              int     `json:"headers"`
		Blocks               int     `json:"blocks"`
		VerificationProgress float64 `json:"verificationProgress"`
	} `json:"nodeInformation,omitempty"`
	PaymentMethodID string `json:"paymentMethodId"`
	Available       bool   `json:"available"`
	Summary         struct {
		Synced          bool      `json:"synced"`
		CurrentHeight   int       `json:"currentHeight"`
		WalletHeight    int       `json:"walletHeight"`
		TargetHeight    int       `json:"targetHeight"`
		UpdatedAt       time.Time `json:"updatedAt"`
		DaemonAvailable bool      `json:"daemonAvailable"`
		WalletAvailable bool      `json:"walletAvailable"`
	} `json:"summary,omitempty"`
}

// A zero Store can be used in development, but there is no "demo mode" because a demo mode could cause confusion whether a payment is real or not.
type Store struct {
	Host          string             `json:"uri"`   // without "/api" and without trailing slash, used for API access and user links
	HostOnion     string             `json:"onion"` // without "/api" and without trailing slash, used for user links only, can be empty
	ID            string             `json:"id"`
	MaxRates      map[string]float64 `json:"maxRates"`   // example: {"XMR": 1000, "BTC": 500000}
	UserAPIKey    string             `json:"userAPIKey"` // must be created in the BTCPay Server user settings (not in the store settings)
	WebhookSecret string             `json:"webhookSecret"`
}

// LoadConfig unmarshals a json config file into a Store.
func LoadConfig(jsonPath string) (Store, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return Store{}, err
	}

	var store Store
	return store, json.Unmarshal(data, &store)
}

func (s Store) doRequest(method string, path string, body io.Reader) (*http.Response, error) {
	if s.Host == "" {
		return nil, errors.New("missing host in btcpay configuration")
	}
	if s.ID == "" {
		return nil, errors.New("missing id in btcpay configuration")
	}
	if s.UserAPIKey == "" {
		return nil, errors.New("missing user API key in btcpay configuration")
	}

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
func (s Store) CheckInvoiceAuth() error {
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
func (s Store) CreateInvoice(req *InvoiceRequest) (*Invoice, error) {
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

func (s Store) GetInvoice(id string) (*Invoice, error) {
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

func (s Store) GetInvoicePaymentMethods(id string) ([]InvoicePaymentMethod, error) {
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
func (s Store) GetServerStatus() (*ServerStatus, error) {
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

// like invoice.CheckoutLink but with onion support
func (s Store) InvoiceCheckoutLink(id string, preferOnion bool) string {
	host := s.Host
	if preferOnion && s.HostOnion != "" {
		host = s.HostOnion
	}
	return fmt.Sprintf("%s/i/%s", host, id)
}

func (s Store) ParseInvoiceWebhook(r *http.Request) (*InvoiceEvent, error) {
	var messageMAC = []byte(strings.TrimPrefix(r.Header.Get("BTCPay-Sig"), "sha256="))
	if len(messageMAC) == 0 {
		return nil, ErrWebhookSig
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	var expectedMAC = []byte(hex.EncodeToString(hmac.New(sha256.New, []byte(s.WebhookSecret)).Sum(body)))
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, ErrWebhookSig // don't leak expectedMAC!
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
	if len(s.MaxRates) > 0 {
		paymentMethods, err := s.GetInvoicePaymentMethods(event.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("getting payment methods from invoice: %w", err)
		}
		for cryptoCode, maxRate := range s.MaxRates {
			if err := ValidateRate(paymentMethods, cryptoCode, maxRate); err != nil {
				return nil, fmt.Errorf("validating rate: %w", err)
			}
		}
	}

	return event, nil
}

// StatusDaemon starts a goroutine which fetches the sync status data every 10 seconds and caches it, and returns a getter function.
func (s Store) StatusDaemon() func() []StatusItem {
	var status []StatusItem
	var lock sync.RWMutex

	go func() {
		for ; true; <-time.Tick(10 * time.Second) {
			serverStatus, err := s.GetServerStatus()
			if err != nil {
				continue
			}

			var s []StatusItem
			for _, syncStatus := range serverStatus.SyncStatuses {
				switch syncStatus.PaymentMethodID {
				case "BTC-CHAIN":
					s = append(s, StatusItem{
						Name:   "BTC",
						Synced: syncStatus.Available && syncStatus.ChainHeight == syncStatus.SyncHeight,
					})
				case "XMR-CHAIN":
					s = append(s, StatusItem{
						Name:   "XMR",
						Synced: syncStatus.Available && syncStatus.Summary.Synced && syncStatus.Summary.DaemonAvailable && syncStatus.Summary.WalletAvailable,
					})
				}
			}

			lock.Lock()
			status = s
			lock.Unlock()
		}
	}()

	return func() []StatusItem {
		lock.RLock()
		defer lock.RUnlock()
		return slices.Clone(status) // don't return original slice
	}
}

type StatusItem struct {
	Name   string
	Synced bool // not just synced but also daemon running, wallet available, etc.
}
