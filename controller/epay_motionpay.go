package controller

import (
	json "encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type EpayCheckoutResponse struct {
	PayLink     string
	PayLinkType string
	QRContent   string
	URL         string
	Params      map[string]string
}

type motionPayMAPIResponse struct {
	Code      json.RawMessage `json:"code"`
	Msg       string          `json:"msg"`
	TradeNo   string          `json:"trade_no"`
	PayURL    string          `json:"payurl"`
	QRCode    string          `json:"qrcode"`
	URLScheme string          `json:"urlscheme"`
}

func isMotionPayGateway(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "motionpay.net" ||
		host == "www.motionpay.net" ||
		host == "aodww.cn" ||
		strings.HasSuffix(host, ".aodww.cn")
}

func motionPayUsesMAPI() bool {
	return isMotionPayGateway(operation_setting.PayAddress)
}

func isMotionPaySuccessCode(raw json.RawMessage) bool {
	code := strings.TrimSpace(string(raw))
	return code == "1" || code == `"1"`
}

func detectMotionPayDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "micromessenger"):
		return "wechat"
	case strings.Contains(ua, "alipayclient"):
		return "alipay"
	case strings.Contains(ua, "qq/"):
		return "qq"
	case strings.Contains(ua, "mobile"), strings.Contains(ua, "android"), strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		return "mobile"
	default:
		return "pc"
	}
}

func normalizeClientIP(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return "127.0.0.1"
	}
	return parsed.String()
}

func isMotionPayCashierLink(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.Contains(strings.ToLower(raw), "/cashier.php")
	}
	return strings.HasSuffix(strings.ToLower(u.Path), "/cashier.php")
}

func isTrustedMotionPayDirectLink(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(u.Scheme)) {
	case "alipays":
		return true
	case "https":
		host := strings.ToLower(strings.TrimSpace(u.Hostname()))
		if host == "qr.alipay.com" {
			return true
		}
		if host == "alipay.com" || strings.HasSuffix(host, ".alipay.com") {
			return true
		}
		if host == "alipayobjects.com" || strings.HasSuffix(host, ".alipayobjects.com") {
			return true
		}
	}
	return false
}

func validateMotionPayCheckoutLink(link string) error {
	link = strings.TrimSpace(link)
	if link == "" {
		return nil
	}
	if isMotionPayCashierLink(link) {
		return fmt.Errorf("current merchant alipay channel returned cashier page instead of direct payment link")
	}
	if !isTrustedMotionPayDirectLink(link) {
		return fmt.Errorf("motionpay returned untrusted payment link")
	}
	return nil
}

func validateMotionPayCheckout(checkout *EpayCheckoutResponse) error {
	if checkout == nil {
		return fmt.Errorf("motionpay checkout is empty")
	}
	if err := validateMotionPayCheckoutLink(checkout.PayLink); err != nil {
		return err
	}
	if err := validateMotionPayCheckoutLink(checkout.QRContent); err != nil {
		return err
	}
	return nil
}

func shouldUseMotionPayMAPI(args *epay.PurchaseArgs) bool {
	if args == nil {
		return false
	}
	// Keep WeChat on the legacy submit.php flow.
	// MotionPay's mapi.php returns a wxjspay URL that is suitable for in-WeChat
	// jumps, but breaks the current desktop QR-code checkout flow.
	return strings.EqualFold(args.Type, "alipay")
}

func motionPayQRCodePreferred(args *epay.PurchaseArgs, device string) bool {
	if args == nil || !strings.EqualFold(args.Type, "alipay") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(device)) {
	case "", "pc":
		return true
	default:
		return false
	}
}

func newMotionPayCheckoutResponse(link, linkType string) *EpayCheckoutResponse {
	link = strings.TrimSpace(link)
	if link == "" {
		return nil
	}
	resp := &EpayCheckoutResponse{
		PayLink:     link,
		PayLinkType: linkType,
	}
	if linkType == "qrcode" {
		resp.QRContent = link
	}
	return resp
}

func pickMotionPayCheckout(candidates ...*EpayCheckoutResponse) *EpayCheckoutResponse {
	for _, candidate := range candidates {
		if candidate != nil && strings.TrimSpace(candidate.PayLink) != "" {
			return candidate
		}
	}
	return nil
}

func requestMotionPayMAPI(c *gin.Context, client *epay.Client, args *epay.PurchaseArgs) (*EpayCheckoutResponse, error) {
	mapiURL := *client.BaseUrl
	mapiURL.Path = path.Join(mapiURL.Path, "/mapi.php")
	device := detectMotionPayDevice(c.Request.UserAgent())

	params := epay.GenerateParams(map[string]string{
		"pid":          client.Config.PartnerID,
		"type":         args.Type,
		"out_trade_no": args.ServiceTradeNo,
		"notify_url":   args.NotifyUrl.String(),
		"return_url":   args.ReturnUrl.String(),
		"name":         args.Name,
		"money":        args.Money,
		"clientip":     normalizeClientIP(c.ClientIP()),
		"device":       device,
	}, client.Config.Key)
	if motionPayQRCodePreferred(args, device) {
		params["rawurl"] = "1"
		params = epay.GenerateParams(params, client.Config.Key)
	}

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}

	req, err := http.NewRequest(http.MethodPost, mapiURL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := service.GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload motionPayMAPIResponse
	if err = common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("decode motionpay response failed: %w", err)
	}
	if !isMotionPaySuccessCode(payload.Code) {
		msg := strings.TrimSpace(payload.Msg)
		if msg == "" {
			msg = "motionpay create payment failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	qrCheckout := newMotionPayCheckoutResponse(payload.QRCode, "qrcode")
	urlCheckout := newMotionPayCheckoutResponse(payload.PayURL, "url")
	schemeCheckout := newMotionPayCheckoutResponse(payload.URLScheme, "url")

	checkout := pickMotionPayCheckout(urlCheckout, schemeCheckout, qrCheckout)
	if motionPayQRCodePreferred(args, device) {
		checkout = pickMotionPayCheckout(qrCheckout, urlCheckout, schemeCheckout)
	}
	if checkout == nil {
		return nil, fmt.Errorf("motionpay returned no usable payment link")
	}
	if err := validateMotionPayCheckout(checkout); err != nil {
		return nil, err
	}
	return checkout, nil
}

func CreateEpayCheckout(c *gin.Context, args *epay.PurchaseArgs) (*EpayCheckoutResponse, error) {
	client := GetEpayClient()
	if client == nil {
		return nil, fmt.Errorf("payment gateway is not configured")
	}

	if motionPayUsesMAPI() && shouldUseMotionPayMAPI(args) {
		checkout, err := requestMotionPayMAPI(c, client, args)
		if err != nil {
			common.SysError(fmt.Sprintf("motionpay mapi request failed: %v", err))
			return nil, err
		}
		return checkout, nil
	}

	uri, params, err := client.Purchase(args)
	if err != nil {
		return nil, err
	}
	return &EpayCheckoutResponse{URL: uri, Params: params}, nil
}

func getEpayMethodDisplayName(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "alipay":
		return "支付宝"
	case "wxpay":
		return "微信支付"
	case "qqpay":
		return "QQ支付"
	default:
		return method
	}
}

func buildEpayCheckoutPayload(checkout *EpayCheckoutResponse, extras gin.H) gin.H {
	data := gin.H{
		"pay_link": checkout.PayLink,
	}
	if strings.TrimSpace(checkout.PayLinkType) != "" {
		data["pay_link_type"] = checkout.PayLinkType
	}
	if strings.TrimSpace(checkout.QRContent) != "" {
		data["qr_content"] = checkout.QRContent
	}
	for key, value := range extras {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}
		data[key] = value
	}
	return data
}
