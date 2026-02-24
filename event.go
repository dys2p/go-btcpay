package btcpay

type EventType string

// An invoice is considered "settled" if it has been "paid" in time (seen on the blockchain before the invoice time expired) and the full amount has been paid and the transaction has been confirmed (got n confirmations on the blockchain, see SpeedPolicy).
const (
	EventInvoiceCreated         EventType = "InvoiceCreated"
	EventInvoiceExpired         EventType = "InvoiceExpired"
	EventInvoiceInvalid         EventType = "InvoiceInvalid"
	EventInvoicePaymentSettled  EventType = "InvoicePaymentSettled"
	EventInvoiceProcessing      EventType = "InvoiceProcessing"
	EventInvoiceReceivedPayment EventType = "InvoiceReceivedPayment"
	EventInvoiceSettled         EventType = "InvoiceSettled"
)

type PaymentStatus string

const (
	PaymentInvalid    PaymentStatus = "Invalid"
	PaymentProcessing PaymentStatus = "Processing"
	PaymentSettled    PaymentStatus = "Settled"
)

// An InvoiceEvent is sent by a webhook.
// You can find your custom OrderID in InvoiceMetadata or use GetInvoice to obtain the full invoice.
type InvoiceEvent struct {
	DeliveryID         string          `json:"deliveryId"`
	InvoiceID          string          `json:"invoiceId"`
	IsRedelivery       bool            `json:"isRedelivery"`
	OriginalDeliveryID string          `json:"originalDeliveryId"`
	StoreID            string          `json:"storeId"`
	Timestamp          int64           `json:"timestamp"`
	Type               EventType       `json:"type"`
	WebhookID          string          `json:"webhookId"`
	InvoiceMetadata    InvoiceMetadata `json:"metadata"`

	// InvoiceInvalid and InvoiceSettled only
	ManuallyMarked bool `json:"manuallyMarked"`

	// InvoiceReceivedPayment and InvoicePaymentSettled only
	AfterExpiration bool     `json:"afterExpiration"` // whether this payment has been sent after the invoice expired
	PaymentMethodID string   `json:"paymentMethodId"` // what payment method was used for this payment
	Payment         struct { // details about the payment
		ID           string        `json:"id"`           // a unique identifier for this payment
		ReceivedDate int           `json:"receivedDate"` // the date the payment was recorded
		Value        string        `json:"value"`        // the value of the payment (in crypto currency)
		Fee          string        `json:"fee"`          // the fee paid for the payment
		Status       PaymentStatus `json:"status"`       // the status of the payment
		Destination  string        `json:"destination"`  // the destination the payment was made to
	} `json:"payment"`

	// InvoiceExpired only
	PartiallyPaid bool `json:"partiallyPaid"`

	// InvoiceProcessing and InvoiceSettled only
	OverPaid bool `json:"overPaid"`

	// fetched by func ParseInvoiceWebhook
	Rate float64 // example: 64392.23
}
