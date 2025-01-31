package btcpay

import (
	"net/http"
	"time"
)

type Store interface {
	CheckInvoiceAuth() error
	CreateInvoice(req *InvoiceRequest) (*Invoice, error)
	CreatePaymentRequest(req *PaymentRequestRequest) (*PaymentRequest, error)
	GetInvoice(id string) (*Invoice, error)
	GetPaymentRequest(id string) (*PaymentRequest, error)
	GetServerStatus() (*ServerStatus, error)
	InvoiceCheckoutLink(id string) string
	InvoiceCheckoutLinkPreferOnion(id string) string
	PaymentRequestLink(id string) string
	PaymentRequestLinkPreferOnion(id string) string
	ProcessWebhook(req *http.Request) (*InvoiceEvent, error)
}

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
