// Package paidsub implements the experimental "Paid Subscriptions" module: a
// client-facing Telegram bot, self-registration, and tariff-based payments.
package paidsub

import "github.com/deposist/s-ui-x/database/model"

// Persistent paid-subscription models live in database/model so central backup
// and restore can include them without importing paidsub and creating a cycle.
type Tariff = model.PaidSubTariff
type PaymentOrder = model.PaidSubPaymentOrder
type Binding = model.PaidSubBinding
type PollCursor = model.PaidSubPollCursor
type InvoiceCancellation = model.PaidSubInvoiceCancellation

const (
	paymentOrderSnapshotVersion       = 1
	paymentOrderLegacyResolvedVersion = 2
)

const (
	StatusPending         = "pending"
	StatusInvoiceCreating = "invoice_creating"
	StatusRecoverable     = "recoverable"
	StatusManualReview    = "manual_review"
	StatusPaid            = "paid"
	StatusFailed          = "failed"
	StatusExpired         = "expired"
	StatusRefunded        = "refunded"
)
