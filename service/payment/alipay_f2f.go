package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

const (
	alipayF2FMethod     = "alipay.trade.precreate"
	alipayF2FFormat     = "JSON"
	alipayF2FCharset    = "utf-8"
	alipayF2FSignType   = "RSA2"
	alipayF2FVersion    = "1.0"
	alipayF2FProduct    = "FACE_TO_FACE_PAYMENT"
	alipayF2FRespKey    = "alipay_trade_precreate_response"
	alipayF2FSandboxURL = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	CallbackStatusOK    = "success"
	CallbackStatusWait  = "pending"
	CallbackStatusFail  = "failed"
)

type AlipayF2FProvider struct{}

type alipayPrecreateResponse struct {
	Response struct {
		Code       string `json:"code"`
		Msg        string `json:"msg"`
		SubCode    string `json:"sub_code"`
		SubMsg     string `json:"sub_msg"`
		OutTradeNo string `json:"out_trade_no"`
		QRCode     string `json:"qr_code"`
	} `json:"alipay_trade_precreate_response"`
	Sign string `json:"sign"`
}

func NewAlipayF2FProvider() *AlipayF2FProvider {
	return &AlipayF2FProvider{}
}

func (*AlipayF2FProvider) Type() string {
	return model.PaymentProviderAlipayF2F
}

func (*AlipayF2FProvider) DisplayName() string {
	if strings.TrimSpace(setting.AlipayF2FDisplayName) != "" {
		return strings.TrimSpace(setting.AlipayF2FDisplayName)
	}
	return "支付宝当面付"
}

func (*AlipayF2FProvider) ValidateConfig() error {
	if !setting.AlipayF2FEnabled {
		return errors.New("支付宝当面付未启用")
	}
	if strings.TrimSpace(setting.AlipayF2FAppId) == "" {
		return errors.New("支付宝 AppID 未配置")
	}
	if strings.TrimSpace(setting.AlipayF2FPrivateKey) == "" {
		return errors.New("支付宝应用私钥未配置")
	}
	if strings.TrimSpace(setting.AlipayF2FPublicKey) == "" {
		return errors.New("支付宝公钥未配置")
	}
	if strings.TrimSpace(setting.AlipayF2FGatewayUrl) == "" {
		return errors.New("支付宝网关未配置")
	}
	if _, err := parseRSAPrivateKey(setting.AlipayF2FPrivateKey); err != nil {
		return fmt.Errorf("支付宝应用私钥无效: %w", err)
	}
	if _, err := parseRSAPublicKey(setting.AlipayF2FPublicKey); err != nil {
		return fmt.Errorf("支付宝公钥无效: %w", err)
	}
	return nil
}

func (p *AlipayF2FProvider) CreatePayment(ctx context.Context, req PaymentCreateContext) (*PaymentCreateResult, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}
	if req.TradeNo == "" {
		return nil, errors.New("订单号为空")
	}
	if req.Amount < 0.01 {
		return nil, errors.New("支付金额过低")
	}
	if err := validatePublicNotifyURL(req.NotifyURL); err != nil {
		return nil, err
	}

	bizContent, err := json.Marshal(map[string]string{
		"out_trade_no": req.TradeNo,
		"total_amount": strconv.FormatFloat(req.Amount, 'f', 2, 64),
		"subject":      req.Subject,
		"body":         req.Body,
		"product_code": alipayF2FProduct,
	})
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"app_id":      strings.TrimSpace(setting.AlipayF2FAppId),
		"method":      alipayF2FMethod,
		"format":      alipayF2FFormat,
		"charset":     alipayF2FCharset,
		"sign_type":   alipayF2FSignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     alipayF2FVersion,
		"notify_url":  req.NotifyURL,
		"biz_content": string(bizContent),
	}

	sign, err := signAlipayParams(params, setting.AlipayF2FPrivateKey)
	if err != nil {
		return nil, err
	}
	params["sign"] = sign

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}

	gatewayURL := strings.TrimSpace(setting.AlipayF2FGatewayUrl)
	if setting.AlipayF2FSandboxEnabled {
		gatewayURL = alipayF2FSandboxURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("支付宝预下单 HTTP 状态异常: %d", resp.StatusCode)
	}

	var parsed alipayPrecreateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("支付宝预下单响应解析失败: %w", err)
	}
	if parsed.Response.Code != "10000" {
		msg := parsed.Response.SubMsg
		if msg == "" {
			msg = parsed.Response.Msg
		}
		if msg == "" {
			msg = "支付宝预下单失败"
		}
		return nil, errors.New(msg)
	}
	if strings.TrimSpace(parsed.Response.QRCode) == "" {
		return nil, errors.New("支付宝未返回二维码")
	}

	return &PaymentCreateResult{
		PaymentProvider: model.PaymentProviderAlipayF2F,
		OutTradeNo:      req.TradeNo,
		QRCode:          parsed.Response.QRCode,
		RawPayload:      string(body),
	}, nil
}

func (*AlipayF2FProvider) VerifyCallback(params map[string]string) error {
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return errors.New("支付宝回调缺少 sign")
	}
	return verifyAlipayParams(params, setting.AlipayF2FPublicKey)
}

func (*AlipayF2FProvider) ParseCallback(params map[string]string) (*PaymentCallbackResult, error) {
	tradeNo := strings.TrimSpace(params["out_trade_no"])
	if tradeNo == "" {
		return nil, errors.New("支付宝回调缺少 out_trade_no")
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(params["total_amount"]), 64)
	if err != nil {
		return nil, errors.New("支付宝回调金额无效")
	}
	status := CallbackStatusWait
	switch strings.TrimSpace(params["trade_status"]) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		status = CallbackStatusOK
	case "":
		status = CallbackStatusFail
	default:
		status = CallbackStatusWait
	}
	return &PaymentCallbackResult{
		TradeNo:         tradeNo,
		ProviderTradeNo: strings.TrimSpace(params["trade_no"]),
		Status:          status,
		PaidAmount:      amount,
		AppID:           strings.TrimSpace(params["app_id"]),
		SellerID:        strings.TrimSpace(params["seller_id"]),
		RawPayload:      common.GetJsonString(params),
	}, nil
}

func signAlipayParams(params map[string]string, privateKey string) (string, error) {
	key, err := parseRSAPrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	unsigned := buildAlipaySignContent(params)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func verifyAlipayParams(params map[string]string, publicKey string) error {
	key, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(params["sign"]))
	if err != nil {
		return err
	}
	unsigned := buildAlipaySignContent(params)
	digest := sha256.Sum256([]byte(unsigned))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature)
}

func buildAlipaySignContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func parseRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, err := pemBlock(raw, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return key, nil
}

func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	block, err := pemBlock(raw, "PUBLIC KEY")
	if err != nil {
		return nil, err
	}
	if parsed, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if key, ok := parsed.(*rsa.PublicKey); ok {
			return key, nil
		}
	}
	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func pemBlock(raw string, blockType string) (*pem.Block, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty key")
	}
	if strings.Contains(trimmed, "-----BEGIN") {
		block, _ := pem.Decode([]byte(trimmed))
		if block == nil {
			return nil, errors.New("invalid PEM")
		}
		return block, nil
	}
	compact := strings.NewReplacer("\r", "", "\n", "", " ", "").Replace(trimmed)
	data, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, err
	}
	return &pem.Block{Type: blockType, Bytes: data}, nil
}

func validatePublicNotifyURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return errors.New("支付宝 notify_url 配置错误")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("支付宝 notify_url 必须使用 http 或 https")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		return errors.New("支付宝 notify_url 不能使用本地域名或本地 IP")
	}
	return nil
}

func ContentTypeIsForm(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data"
}

func ReadFormOrQueryParams(req *http.Request) map[string]string {
	values := url.Values{}
	for key, vals := range req.URL.Query() {
		for _, value := range vals {
			values.Add(key, value)
		}
	}
	if req.Method == http.MethodPost && ContentTypeIsForm(req.Header.Get("Content-Type")) {
		_ = req.ParseForm()
		for key, vals := range req.PostForm {
			for _, value := range vals {
				values.Set(key, value)
			}
		}
		return MergeCallbackParams(values)
	}
	if req.Method == http.MethodPost && req.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(req.Body, 1024*1024))
		req.Body = io.NopCloser(bytes.NewReader(body))
		form, err := url.ParseQuery(string(body))
		if err == nil {
			for key, vals := range form {
				for _, value := range vals {
					values.Set(key, value)
				}
			}
		}
	}
	return MergeCallbackParams(values)
}
