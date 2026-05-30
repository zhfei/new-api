package setting

const (
	AlipayF2FSubject = "启宝扫码点餐订单"
	AlipayF2FBody    = "线下餐饮扫码点餐服务"
)

var (
	AlipayF2FEnabled               bool
	AlipayF2FAppId                 string
	AlipayF2FPrivateKey            string
	AlipayF2FPublicKey             string
	AlipayF2FGatewayUrl            string = "https://openapi.alipay.com/gateway.do"
	AlipayF2FSandboxEnabled        bool
	AlipayF2FTopUpNotifyUrl        string
	AlipayF2FTopUpReturnUrl        string
	AlipayF2FSubscriptionNotifyUrl string
	AlipayF2FSubscriptionReturnUrl string
	AlipayF2FSellerId              string
	AlipayF2FMinTopUp              int    = 1
	AlipayF2FDisplayName           string = "支付宝当面付"
)
