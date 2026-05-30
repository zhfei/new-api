package controller

import (
	"strings"
	"testing"

	paymentservice "github.com/QuantumNous/new-api/service/payment"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
)

func configureAlipayF2FCallbackForTest(t *testing.T) {
	t.Helper()
	oldAppID := setting.AlipayF2FAppId
	oldSellerID := setting.AlipayF2FSellerId
	t.Cleanup(func() {
		setting.AlipayF2FAppId = oldAppID
		setting.AlipayF2FSellerId = oldSellerID
	})
	setting.AlipayF2FAppId = "app-1"
	setting.AlipayF2FSellerId = "seller-1"
}

func TestValidateAlipayF2FCallback(t *testing.T) {
	configureAlipayF2FCallbackForTest(t)
	base := &paymentservice.PaymentCallbackResult{
		TradeNo:    "USR1NOabc",
		PaidAmount: 12.34,
		AppID:      "app-1",
		SellerID:   "seller-1",
	}
	require.NoError(t, validateAlipayF2FCallback(base, 12.34))

	amountMismatch := *base
	amountMismatch.PaidAmount = 12.35
	require.ErrorContains(t, validateAlipayF2FCallback(&amountMismatch, 12.34), "金额不匹配")

	appMismatch := *base
	appMismatch.AppID = "app-2"
	require.ErrorContains(t, validateAlipayF2FCallback(&appMismatch, 12.34), "AppID 不匹配")

	sellerMismatch := *base
	sellerMismatch.SellerID = "seller-2"
	require.ErrorContains(t, validateAlipayF2FCallback(&sellerMismatch, 12.34), "SellerID 不匹配")
}

func TestAlipayF2FQRCodeDataURL(t *testing.T) {
	dataURL := buildAlipayF2FQRCodeDataURL("https://qr.alipay.example/test")
	require.True(t, strings.HasPrefix(dataURL, "data:image/png;base64,"))
	require.Empty(t, buildAlipayF2FQRCodeDataURL(""))
}

func TestMergeAlipayF2FProviderPayload(t *testing.T) {
	merged := mergeAlipayF2FProviderPayload(
		`{"qr_code":"https://qr.alipay.example/test","create_result":{"out_trade_no":"USR1NOabc"}}`,
		`{"out_trade_no":"USR1NOabc","trade_status":"TRADE_SUCCESS"}`,
	)
	payload := parseAlipayF2FPayload(merged)
	require.Equal(t, "https://qr.alipay.example/test", payload["qr_code"])
	require.Contains(t, payload, "create_result")
	require.Contains(t, payload, "callback_result")
}
