package controller

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	paymentservice "github.com/QuantumNous/new-api/service/payment"
	"github.com/QuantumNous/new-api/setting"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func alipayF2FProvider() *paymentservice.AlipayF2FProvider {
	return paymentservice.NewAlipayF2FProvider()
}

func resolveAlipayF2FURL(configured string, fallbackPath string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	return strings.TrimRight(service.GetCallbackAddress(), "/") + fallbackPath
}

func alipayF2FTopUpNotifyURL() string {
	return resolveAlipayF2FURL(setting.AlipayF2FTopUpNotifyUrl, "/api/user/alipay-f2f/notify")
}

func alipayF2FTopUpReturnURL() string {
	return resolveAlipayF2FURL(setting.AlipayF2FTopUpReturnUrl, "/api/user/alipay-f2f/return")
}

func alipayF2FSubscriptionNotifyURL() string {
	return resolveAlipayF2FURL(setting.AlipayF2FSubscriptionNotifyUrl, "/api/subscription/alipay-f2f/notify")
}

func alipayF2FSubscriptionReturnURL() string {
	return resolveAlipayF2FURL(setting.AlipayF2FSubscriptionReturnUrl, "/api/subscription/alipay-f2f/return")
}

func buildAlipayF2FResponse(result *paymentservice.PaymentCreateResult, statusPath string, pagePath string) gin.H {
	if result == nil {
		return gin.H{}
	}
	if result.QRCodeDataURL == "" {
		result.QRCodeDataURL = buildAlipayF2FQRCodeDataURL(result.QRCode)
	}
	result.StatusURL = statusPath + "?out_trade_no=" + result.OutTradeNo
	result.PaymentPageURL = pagePath + "?out_trade_no=" + result.OutTradeNo
	return gin.H{
		"payment_provider": result.PaymentProvider,
		"out_trade_no":     result.OutTradeNo,
		"qr_code":          result.QRCode,
		"qr_code_data_url": result.QRCodeDataURL,
		"status_url":       result.StatusURL,
		"payment_page_url": result.PaymentPageURL,
	}
}

func buildAlipayF2FQRCodeDataURL(qrCode string) string {
	qrCode = strings.TrimSpace(qrCode)
	if qrCode == "" {
		return ""
	}
	code, err := qr.Encode(qrCode, qr.M, qr.Auto)
	if err != nil {
		return ""
	}
	scaled, err := barcode.Scale(code, 256, 256)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func parseAlipayF2FPayload(raw string) map[string]any {
	var payload map[string]any
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func mergeAlipayF2FProviderPayload(existing string, callbackRaw string) string {
	payload := parseAlipayF2FPayload(existing)
	if payload == nil {
		payload = map[string]any{}
	}
	if strings.TrimSpace(callbackRaw) != "" {
		var callback map[string]any
		if err := common.UnmarshalJsonStr(callbackRaw, &callback); err == nil {
			payload["callback_result"] = callback
		} else {
			payload["callback_result"] = callbackRaw
		}
	}
	return common.GetJsonString(payload)
}

func alipayF2FPayloadQRCode(raw string) string {
	payload := parseAlipayF2FPayload(raw)
	if qr, ok := payload["qr_code"].(string); ok {
		return qr
	}
	if data, ok := payload["create_result"].(map[string]any); ok {
		if qr, ok := data["qr_code"].(string); ok {
			return qr
		}
	}
	return ""
}

func validateAlipayF2FCallback(result *paymentservice.PaymentCallbackResult, orderMoney float64) error {
	if result == nil {
		return fmt.Errorf("支付宝回调为空")
	}
	if result.AppID != strings.TrimSpace(setting.AlipayF2FAppId) {
		return fmt.Errorf("支付宝 AppID 不匹配")
	}
	if strings.TrimSpace(setting.AlipayF2FSellerId) != "" && result.SellerID != strings.TrimSpace(setting.AlipayF2FSellerId) {
		return fmt.Errorf("支付宝 SellerID 不匹配")
	}
	expected := decimal.NewFromFloat(orderMoney).Round(2)
	actual := decimal.NewFromFloat(result.PaidAmount).Round(2)
	if !expected.Equal(actual) {
		return fmt.Errorf("支付宝回调金额不匹配 expected=%s actual=%s", expected.StringFixed(2), actual.StringFixed(2))
	}
	return nil
}

func writeAlipayF2FNotify(c *gin.Context, ok bool) {
	if ok {
		_, _ = c.Writer.Write([]byte("success"))
		return
	}
	_, _ = c.Writer.Write([]byte("fail"))
}

func alipayF2FReturn(c *gin.Context) {
	c.Redirect(http.StatusFound, paymentReturnPath("/console/topup?pay=pending"))
}

func AlipayF2FTopUpReturn(c *gin.Context) {
	alipayF2FReturn(c)
}

func AlipayF2FSubscriptionReturn(c *gin.Context) {
	alipayF2FReturn(c)
}

func renderAlipayF2FQRCodePage(c *gin.Context, title string, tradeNo string, qrCode string, statusURL string) {
	qrImage := buildAlipayF2FQRCodeDataURL(qrCode)
	qrHTML := `<div class="empty">二维码内容为空，请返回重新发起支付。</div>`
	if qrImage != "" {
		qrHTML = `<img class="qr-img" src="` + html.EscapeString(qrImage) + `" alt="支付宝支付二维码">`
	} else if strings.TrimSpace(qrCode) != "" {
		qrHTML = `<div class="qr-text">` + html.EscapeString(qrCode) + `</div>`
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` +
		html.EscapeString(title) +
		`</title><style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f7f9;margin:0;padding:32px;color:#111}.card{max-width:560px;margin:0 auto;background:#fff;border-radius:18px;padding:28px;box-shadow:0 20px 60px rgba(15,23,42,.12);text-align:center}.qr{display:flex;justify-content:center;margin-top:18px}.qr-img{width:256px;height:256px;border:12px solid #fff;box-shadow:0 10px 30px rgba(15,23,42,.1)}.qr-text,.empty{word-break:break-all;background:#f8fafc;border:1px solid #e2e8f0;border-radius:12px;padding:16px;text-align:left}.muted{color:#64748b;font-size:14px}</style></head><body><div class="card"><h2>` +
		html.EscapeString(title) +
		`</h2><p class="muted">请使用支付宝扫码支付。订单号：` +
		html.EscapeString(tradeNo) +
		`</p><div class="qr">` +
		qrHTML +
		`</div><p class="muted">支付完成后可返回原页面，系统会自动刷新状态。</p><p class="muted">支付状态接口：` +
		html.EscapeString(statusURL) +
		`</p></div></body></html>`))
}

func AlipayF2FTopUpQRCode(c *gin.Context) {
	tradeNo := c.Query("out_trade_no")
	userId := c.GetInt("id")
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != userId || topUp.PaymentProvider != model.PaymentProviderAlipayF2F {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	renderAlipayF2FQRCodePage(c, "支付宝当面付", tradeNo, alipayF2FPayloadQRCode(topUp.ProviderPayload), "/api/user/alipay-f2f/status?out_trade_no="+tradeNo)
}

func AlipayF2FSubscriptionQRCode(c *gin.Context) {
	tradeNo := c.Query("out_trade_no")
	userId := c.GetInt("id")
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil || order.UserId != userId || order.PaymentProvider != model.PaymentProviderAlipayF2F {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	renderAlipayF2FQRCodePage(c, "支付宝当面付", tradeNo, alipayF2FPayloadQRCode(order.ProviderPayload), "/api/subscription/alipay-f2f/status?out_trade_no="+tradeNo)
}

func AlipayF2FTopUpStatus(c *gin.Context) {
	tradeNo := c.Query("out_trade_no")
	userId := c.GetInt("id")
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != userId || topUp.PaymentProvider != model.PaymentProviderAlipayF2F {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	common.ApiSuccess(c, gin.H{
		"out_trade_no": tradeNo,
		"status":       topUp.Status,
		"amount":       topUp.Amount,
		"money":        strconv.FormatFloat(topUp.Money, 'f', 2, 64),
	})
}

func AlipayF2FSubscriptionStatus(c *gin.Context) {
	tradeNo := c.Query("out_trade_no")
	userId := c.GetInt("id")
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil || order.UserId != userId || order.PaymentProvider != model.PaymentProviderAlipayF2F {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	common.ApiSuccess(c, gin.H{
		"out_trade_no": tradeNo,
		"status":       order.Status,
		"money":        strconv.FormatFloat(order.Money, 'f', 2, 64),
	})
}
