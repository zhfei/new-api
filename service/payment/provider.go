package payment

import (
	"context"
	"net/url"
)

type PaymentProvider interface {
	Type() string
	DisplayName() string
	ValidateConfig() error
	CreatePayment(ctx context.Context, req PaymentCreateContext) (*PaymentCreateResult, error)
	VerifyCallback(params map[string]string) error
	ParseCallback(params map[string]string) (*PaymentCallbackResult, error)
}

type PaymentCreateContext struct {
	TradeNo   string
	Amount    float64
	Subject   string
	Body      string
	NotifyURL string
	ReturnURL string
}

type PaymentCreateResult struct {
	PaymentProvider string `json:"payment_provider"`
	OutTradeNo      string `json:"out_trade_no"`
	QRCode          string `json:"qr_code"`
	QRCodeDataURL   string `json:"qr_code_data_url,omitempty"`
	PaymentPageURL  string `json:"payment_page_url,omitempty"`
	StatusURL       string `json:"status_url,omitempty"`
	RawPayload      string `json:"raw_payload"`
}

type PaymentCallbackResult struct {
	TradeNo         string
	ProviderTradeNo string
	Status          string
	PaidAmount      float64
	AppID           string
	SellerID        string
	RawPayload      string
}

func MergeCallbackParams(values url.Values) map[string]string {
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	return params
}
