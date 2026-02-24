package btcpay

import (
	"fmt"
	"strconv"
)

const (
	InvoiceNew        string = "New"
	InvoiceProcessing string = "Processing"
	InvoiceExpired    string = "Expired"
	InvoiceInvalid    string = "Invalid"
	InvoiceSettled    string = "Settled"
)

// SpeedPolicy defines when an invoice is considered confirmed.
type SpeedPolicy string

const (
	HighSpeed      SpeedPolicy = "HighSpeed"      // consider the invoice confirmed when the payment transaction has >= 0 confirmations (i.e. as soon as it is visible on the blockchain)
	MediumSpeed    SpeedPolicy = "MediumSpeed"    // consider the invoice confirmed when the payment transaction has >= 1 confirmation
	LowMediumSpeed SpeedPolicy = "LowMediumSpeed" // consider the invoice confirmed when the payment transaction has >= 2 confirmations
	LowSpeed       SpeedPolicy = "LowSpeed"       // consider the invoice confirmed when the payment transaction has >= 6 confirmations
)

type Invoice struct {
	InvoiceRequest
	ID                   string `json:"id"`
	CheckoutLink         string `json:"checkoutLink"`
	CreatedTime          int64  `json:"createdTime"`
	ExpirationTime       int64  `json:"expirationTime"`
	MonitoringExpiration int64  `json:"monitoringExpiration"`
	Status               string `json:"status"`
	AdditionalStatus     string `json:"additionalStatus"`
}

type InvoiceRequest struct {
	Amount          float64 `json:"amount,string"`
	Currency        string  `json:"currency"`
	InvoiceMetadata `json:"metadata,omitempty"`
	InvoiceCheckout `json:"checkout,omitempty"`
}

type InvoiceMetadata struct {
	OrderID string `json:"orderId,omitempty"` // OrderID is stored in the invoice list of your BTCPay server. If the invoice has been created through a payment request, this is the ID of the payment request.
}

type InvoiceCheckout struct {
	SpeedPolicy       SpeedPolicy `json:"speedPolicy,omitempty"` // default: store setting
	PaymentMethods    []string    `json:"paymentMethods,omitempty"`
	ExpirationMinutes int         `json:"expirationMinutes,omitempty"` // refers to the "paid" state, when the transaction becomes visible on the blockchain
	MonitoringMinutes int         `json:"monitoringMinutes,omitempty"`
	PaymentTolerance  float64     `json:"paymentTolerance,omitempty"`
	RedirectURL       string      `json:"redirectURL,omitempty"`     // RedirectURL is stored in the invoice list of your BTCPay server and used as href behind OrderID.
	DefaultLanguage   string      `json:"defaultLanguage,omitempty"` // see https://github.com/btcpayserver/btcpayserver/tree/master/BTCPayServer/wwwroot/locales
}

type InvoicePaymentMethod struct {
	PaymentMethodID   string `json:"paymentMethodId"` // example: "XMR-CHAIN"
	Currency          string `json:"currency"`        // example: "XMR"
	Destination       string `json:"destination"`
	PaymentLink       string `json:"paymentLink"`
	Rate              string `json:"rate"`              // example: "122.7738548555"
	PaymentMethodPaid string `json:"paymentMethodPaid"` // example: "0.03665275"
	TotalPaid         string `json:"totalPaid"`         // Total invoice payment, converted into this currency. This is greater than zero even if there is no payment in this crypto. Be careful!
	Due               string `json:"due"`               // example: "0"
	Amount            string `json:"amount"`            // Some amount, converted into this currency. This is greater than zero even if there is no payment in this crypto. Be careful!
	PaymentMethodFee  string `json:"paymentMethodFee"`
	Payments          []struct {
		ID           string `json:"id"`
		ReceivedDate int    `json:"receivedDate"` // unix timestamp
		Value        string `json:"value"`        // example: "0.036652750000"
		Fee          string `json:"fee"`          // example: "0.0000000003"
		Status       string `json:"status"`       // example: "Settled"
		Destination  string `json:"destination"`
	} `json:"payments"`
	Activated      bool `json:"activated"`
	AdditionalData struct {
		KeyPath              string `json:"keyPath"`
		PayjoinEnabled       bool   `json:"payjoinEnabled"`
		AccountDerivation    string `json:"accountDerivation"`
		RecommendedFeeRate   string `json:"recommendedFeeRate"`
		PaymentMethodFeeRate string `json:"paymentMethodFeeRate"`
	} `json:"additionalData"`
}

// ValidateRates returns an error if the exchange rate for the given currency is above the max rate.
func ValidateRate(methods []InvoicePaymentMethod, currency string, maxRate float64) error {
	for _, method := range methods {
		if method.Currency == currency {
			rate, err := strconv.ParseFloat(method.Rate, 64)
			if err != nil {
				return err
			}
			if rate > maxRate {
				return fmt.Errorf("%s rate %.2f exceeds max rate %.2f", method.Currency, rate, maxRate)
			}
		}
	}
	return nil
}
