package payment

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

func testAlipayF2FKeys(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(privateDER), base64.StdEncoding.EncodeToString(publicDER)
}

func configureAlipayF2FForTest(t *testing.T, gateway string) {
	t.Helper()
	privateKey, publicKey := testAlipayF2FKeys(t)
	oldEnabled := setting.AlipayF2FEnabled
	oldAppID := setting.AlipayF2FAppId
	oldPrivateKey := setting.AlipayF2FPrivateKey
	oldPublicKey := setting.AlipayF2FPublicKey
	oldGatewayURL := setting.AlipayF2FGatewayUrl
	oldSandbox := setting.AlipayF2FSandboxEnabled
	oldDisplayName := setting.AlipayF2FDisplayName
	t.Cleanup(func() {
		setting.AlipayF2FEnabled = oldEnabled
		setting.AlipayF2FAppId = oldAppID
		setting.AlipayF2FPrivateKey = oldPrivateKey
		setting.AlipayF2FPublicKey = oldPublicKey
		setting.AlipayF2FGatewayUrl = oldGatewayURL
		setting.AlipayF2FSandboxEnabled = oldSandbox
		setting.AlipayF2FDisplayName = oldDisplayName
	})
	setting.AlipayF2FEnabled = true
	setting.AlipayF2FAppId = "test-app-id"
	setting.AlipayF2FPrivateKey = privateKey
	setting.AlipayF2FPublicKey = publicKey
	setting.AlipayF2FGatewayUrl = gateway
	setting.AlipayF2FSandboxEnabled = false
	setting.AlipayF2FDisplayName = "支付宝当面付"
}

func TestAlipayF2FSignAndVerifyRoundTrip(t *testing.T) {
	privateKey, publicKey := testAlipayF2FKeys(t)
	params := map[string]string{
		"app_id":        "test-app-id",
		"biz_content":   `{"out_trade_no":"T001","total_amount":"1.00"}`,
		"charset":       "utf-8",
		"method":        "alipay.trade.precreate",
		"sign_type":     "RSA2",
		"empty_ignored": "",
	}
	sign, err := signAlipayParams(params, privateKey)
	if err != nil {
		t.Fatalf("signAlipayParams failed: %v", err)
	}
	params["sign"] = sign
	if err := verifyAlipayParams(params, publicKey); err != nil {
		t.Fatalf("verifyAlipayParams failed: %v", err)
	}
	params["biz_content"] = `{"out_trade_no":"T001","total_amount":"2.00"}`
	if err := verifyAlipayParams(params, publicKey); err == nil {
		t.Fatal("verifyAlipayParams should reject tampered params")
	}
}

func TestAlipayF2FParseCallback(t *testing.T) {
	provider := NewAlipayF2FProvider()
	result, err := provider.ParseCallback(map[string]string{
		"out_trade_no": "USR1NOabc",
		"trade_no":     "202605302200",
		"trade_status": "TRADE_SUCCESS",
		"total_amount": "12.34",
		"app_id":       "test-app-id",
		"seller_id":    "seller-1",
	})
	if err != nil {
		t.Fatalf("ParseCallback failed: %v", err)
	}
	if result.TradeNo != "USR1NOabc" || result.ProviderTradeNo != "202605302200" {
		t.Fatalf("unexpected trade fields: %+v", result)
	}
	if result.Status != CallbackStatusOK {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.PaidAmount != 12.34 {
		t.Fatalf("unexpected amount: %v", result.PaidAmount)
	}
}

func TestAlipayF2FValidatePublicNotifyURL(t *testing.T) {
	invalid := []string{
		"",
		"ftp://example.com/notify",
		"http://localhost/notify",
		"http://127.0.0.1/notify",
		"http://0.0.0.0/notify",
		"http://[::1]/notify",
	}
	for _, raw := range invalid {
		if err := validatePublicNotifyURL(raw); err == nil {
			t.Fatalf("validatePublicNotifyURL should reject %q", raw)
		}
	}
	if err := validatePublicNotifyURL("https://pay.example.com/api/user/alipay-f2f/notify"); err != nil {
		t.Fatalf("validatePublicNotifyURL should accept public https url: %v", err)
	}
}

func TestAlipayF2FCreatePayment(t *testing.T) {
	var received url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm failed: %v", err)
		}
		received = r.PostForm
		bizContent := received.Get("biz_content")
		var biz map[string]string
		if err := json.Unmarshal([]byte(bizContent), &biz); err != nil {
			t.Fatalf("invalid biz_content: %v", err)
		}
		amount, err := strconv.ParseFloat(biz["total_amount"], 64)
		if err != nil || amount != 9.99 {
			t.Fatalf("unexpected total_amount: %q", biz["total_amount"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"alipay_trade_precreate_response":{"code":"10000","msg":"Success","out_trade_no":"USR1NOabc","qr_code":"https://qr.alipay.example/test"},"sign":"ignored"}`)
	}))
	defer server.Close()

	configureAlipayF2FForTest(t, server.URL)
	result, err := NewAlipayF2FProvider().CreatePayment(context.Background(), PaymentCreateContext{
		TradeNo:   "USR1NOabc",
		Amount:    9.99,
		Subject:   "启宝扫码点餐订单",
		Body:      "线下餐饮扫码点餐服务",
		NotifyURL: "https://pay.example.com/api/user/alipay-f2f/notify",
	})
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}
	if result.PaymentProvider != model.PaymentProviderAlipayF2F {
		t.Fatalf("unexpected provider: %s", result.PaymentProvider)
	}
	if result.OutTradeNo != "USR1NOabc" || result.QRCode != "https://qr.alipay.example/test" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if received.Get("sign") == "" {
		t.Fatal("CreatePayment should send RSA2 sign")
	}
	if received.Get("method") != alipayF2FMethod || received.Get("sign_type") != alipayF2FSignType {
		t.Fatalf("unexpected alipay params: %v", received)
	}
}
