package controller

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

const (
	PaymentMethodNOWPayments       = "nowpayments"
	paymentMethodNOWPaymentsPrefix = "nowpayments_"
	nowPaymentsSignatureHeader     = "x-nowpayments-sig"
	nowPaymentsWebhookMaxBody      = 1 << 20
	nowPaymentsMaxAmount           = 10000
	nowPaymentsPricingModeFiat     = "fiat"
	nowPaymentsPricingModeCrypto   = "crypto"
)

type NOWPaymentsQuoteRequest struct {
	Amount        int64  `json:"amount"`
	PricingMode   string `json:"pricing_mode"`
	PayCurrency   string `json:"pay_currency"`
	PaymentMethod string `json:"payment_method"`
}

type NOWPaymentsWebhookPayload struct {
	PaymentID     any    `json:"payment_id"`
	OrderID       string `json:"order_id"`
	PaymentStatus string `json:"payment_status"`
}

type nowPaymentsQuoteResult struct {
	PricingMode   string
	PayCurrency   string
	TopUpAmount   int64
	PriceAmount   float64
	PriceCurrency string
	MinAmount     float64
	MeetsMinimum  bool
}

func RequestNOWPaymentsQuote(c *gin.Context) {
	var req NOWPaymentsQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "请求参数错误"})
		return
	}

	result, err := buildNOWPaymentsQuote(c.GetInt("id"), &req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pricing_mode":   result.PricingMode,
			"pay_currency":   result.PayCurrency,
			"price_amount":   result.PriceAmount,
			"price_currency": result.PriceCurrency,
			"minimum_amount": result.MinAmount,
			"meets_minimum":  result.MeetsMinimum,
			"credit_amount":  result.PriceAmount,
		},
	})
}

func RequestNOWPaymentsPay(c *gin.Context) {
	var req NOWPaymentsQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "请求参数错误"})
		return
	}

	result, err := buildNOWPaymentsQuote(c.GetInt("id"), &req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	if !result.MeetsMinimum {
		c.JSON(http.StatusOK, gin.H{
			"message": "error",
			"data":    fmt.Sprintf("当前金额低于该网络最小支付金额 %.2f", result.MinAmount),
		})
		return
	}

	serverAddress := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	callbackAddress := strings.TrimRight(strings.TrimSpace(service.GetCallbackAddress()), "/")
	if serverAddress == "" || callbackAddress == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "服务器地址或回调地址未配置"})
		return
	}

	client := service.NewNOWPaymentsClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "NOWPayments 未配置或未启用"})
		return
	}

	userID := c.GetInt("id")
	reference := fmt.Sprintf("nowpayments-ref-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(4))
	tradeNo := "ref_" + common.Sha1([]byte(reference))

	topUp := &model.TopUp{
		UserId:        userID,
		Amount:        result.TopUpAmount,
		Money:         result.PriceAmount,
		TradeNo:       tradeNo,
		PaymentMethod: getNOWPaymentsPaymentMethod(result.PayCurrency),
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err = topUp.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建充值订单失败"})
		return
	}

	invoice, err := client.CreateInvoice(&service.NOWPaymentsCreateInvoiceRequest{
		PriceAmount:      result.PriceAmount,
		PriceCurrency:    result.PriceCurrency,
		PayCurrency:      result.PayCurrency,
		OrderID:          tradeNo,
		OrderDescription: fmt.Sprintf("new-api nowpayments %s %s", result.PricingMode, result.PayCurrency),
		IPNCallbackURL:   callbackAddress + "/api/nowpayments/webhook",
		SuccessURL:       serverAddress + "/console/topup",
		CancelURL:        serverAddress + "/console/topup",
	})
	if err != nil {
		log.Printf("create nowpayments invoice failed: trade_no=%s err=%v", tradeNo, err)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起 NOWPayments 支付失败"})
		return
	}
	if strings.TrimSpace(invoice.InvoiceURL) == "" {
		log.Printf("nowpayments invoice missing invoice_url: trade_no=%s invoice_id=%v", tradeNo, invoice.ID)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "NOWPayments 支付链接不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": invoice.InvoiceURL,
		},
	})
}

func NOWPaymentsWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, nowPaymentsWebhookMaxBody+1))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if len(body) > nowPaymentsWebhookMaxBody {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}
	if !service.VerifyNOWPaymentsSignature(body, c.GetHeader(nowPaymentsSignatureHeader)) {
		log.Printf("nowpayments webhook signature verify failed")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var event NOWPaymentsWebhookPayload
	if err = common.Unmarshal(body, &event); err != nil {
		log.Printf("decode nowpayments webhook failed: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	client := service.NewNOWPaymentsClient()
	if client == nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	paymentID := service.FormatNOWPaymentsPaymentID(event.PaymentID)
	if paymentID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	payment, err := client.GetPayment(paymentID)
	if err != nil {
		log.Printf("fetch nowpayments payment failed: payment_id=%s err=%v", paymentID, err)
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	tradeNo := strings.TrimSpace(payment.OrderID)
	if tradeNo == "" {
		tradeNo = strings.TrimSpace(event.OrderID)
	}
	if tradeNo == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err = validateNOWPaymentsPayment(topUp, payment); err != nil {
		log.Printf("nowpayments payment validation failed: trade_no=%s payment_id=%s err=%v", tradeNo, paymentID, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch normalizeNOWPaymentsPaymentStatus(payment.PaymentStatus) {
	case common.TopUpStatusSuccess:
		if topUp.Status == common.TopUpStatusSuccess {
			c.Status(http.StatusOK)
			return
		}
		if topUp.Status != common.TopUpStatusPending {
			log.Printf("ignore finished nowpayments webhook for non-pending order: trade_no=%s status=%s", tradeNo, topUp.Status)
			break
		}
		if err = model.CompleteTopUpByMoney(tradeNo, nil); err != nil {
			log.Printf("complete nowpayments topup failed: trade_no=%s err=%v", tradeNo, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("nowpayments topup completed: trade_no=%s payment_id=%s", tradeNo, paymentID)
	case common.TopUpStatusExpired:
		if err = model.ExpireTopUp(tradeNo); err != nil {
			log.Printf("expire nowpayments topup failed: trade_no=%s err=%v", tradeNo, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("nowpayments topup expired: trade_no=%s payment_id=%s", tradeNo, paymentID)
	default:
		log.Printf("ignore nowpayments webhook: trade_no=%s payment_id=%s status=%s", tradeNo, paymentID, payment.PaymentStatus)
	}

	c.Status(http.StatusOK)
}

func buildNOWPaymentsQuote(userID int, req *NOWPaymentsQuoteRequest) (*nowPaymentsQuoteResult, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数错误")
	}
	if req.PaymentMethod != PaymentMethodNOWPayments {
		return nil, fmt.Errorf("不支持的支付方式")
	}
	if !service.IsNOWPaymentsEnabled() {
		return nil, fmt.Errorf("NOWPayments 未配置或未启用")
	}
	if !service.IsNOWPaymentsModeEnabled(req.PricingMode) {
		return nil, fmt.Errorf("当前定价模式未启用")
	}

	network, ok := service.GetNOWPaymentsNetworkByCode(req.PayCurrency)
	if !ok {
		return nil, fmt.Errorf("不支持的 USDT 网络")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("充值数量必须大于 0")
	}
	if req.Amount > nowPaymentsMaxAmount {
		return nil, fmt.Errorf("充值数量不能大于 %d", nowPaymentsMaxAmount)
	}

	client := service.NewNOWPaymentsClient()
	if client == nil {
		return nil, fmt.Errorf("NOWPayments 未配置或未启用")
	}

	result := &nowPaymentsQuoteResult{
		PricingMode: strings.ToLower(strings.TrimSpace(req.PricingMode)),
		PayCurrency: network.Code,
	}

	switch result.PricingMode {
	case nowPaymentsPricingModeFiat:
		if req.Amount < getMinTopup() {
			return nil, fmt.Errorf("充值数量不能小于 %d", getMinTopup())
		}
		group, err := model.GetUserGroup(userID, true)
		if err != nil {
			return nil, fmt.Errorf("获取用户分组失败")
		}
		priceAmount, err := getPayMoneyUSD(req.Amount, group)
		if err != nil {
			return nil, err
		}
		if priceAmount <= 0.01 {
			return nil, fmt.Errorf("充值金额过低")
		}
		result.TopUpAmount = normalizeNOWPaymentsFiatTopUpAmount(req.Amount)
		result.PriceAmount = priceAmount
		result.PriceCurrency = "usd"
		minResp, err := client.GetMinAmount("usd", result.PayCurrency)
		if err != nil {
			return nil, fmt.Errorf("获取 NOWPayments 最小金额失败")
		}
		result.MinAmount = float64(minResp.MinAmount)
		result.MeetsMinimum = decimal.NewFromFloat(priceAmount).GreaterThanOrEqual(decimal.NewFromFloat(float64(minResp.MinAmount)))
	case nowPaymentsPricingModeCrypto:
		result.TopUpAmount = req.Amount
		result.PriceAmount = float64(req.Amount)
		result.PriceCurrency = result.PayCurrency
		minResp, err := client.GetMinAmount(result.PayCurrency, result.PayCurrency)
		if err != nil {
			return nil, fmt.Errorf("获取 NOWPayments 最小金额失败")
		}
		result.MinAmount = float64(minResp.MinAmount)
		result.MeetsMinimum = decimal.NewFromFloat(result.PriceAmount).GreaterThanOrEqual(decimal.NewFromFloat(float64(minResp.MinAmount)))
	default:
		return nil, fmt.Errorf("不支持的定价模式")
	}

	return result, nil
}

func normalizeNOWPaymentsFiatTopUpAmount(amount int64) int64 {
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return amount
	}
	dAmount := decimal.NewFromInt(amount)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	return dAmount.Div(dQuotaPerUnit).IntPart()
}

func getNOWPaymentsPaymentMethod(payCurrency string) string {
	return paymentMethodNOWPaymentsPrefix + strings.ToLower(strings.TrimSpace(payCurrency))
}

func validateNOWPaymentsPayment(topUp *model.TopUp, payment *service.NOWPaymentsPaymentResponse) error {
	if topUp == nil {
		return fmt.Errorf("topup is nil")
	}
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}
	if !strings.HasPrefix(topUp.PaymentMethod, paymentMethodNOWPaymentsPrefix) {
		return fmt.Errorf("payment method mismatch: %s", topUp.PaymentMethod)
	}
	if strings.TrimSpace(payment.OrderID) != topUp.TradeNo {
		return fmt.Errorf("trade no mismatch")
	}

	expectedPayCurrency := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(topUp.PaymentMethod)), paymentMethodNOWPaymentsPrefix)
	if strings.ToLower(strings.TrimSpace(payment.PayCurrency)) != expectedPayCurrency {
		return fmt.Errorf("pay currency mismatch")
	}

	priceCurrency := strings.ToLower(strings.TrimSpace(payment.PriceCurrency))
	if priceCurrency != "usd" && priceCurrency != expectedPayCurrency {
		return fmt.Errorf("price currency mismatch")
	}

	expectedAmount := decimal.NewFromFloat(topUp.Money).Round(8)
	actualAmount := decimal.NewFromFloat(float64(payment.PriceAmount)).Round(8)
	if !actualAmount.Equal(expectedAmount) {
		return fmt.Errorf("price amount mismatch")
	}
	return nil
}

func normalizeNOWPaymentsPaymentStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "finished":
		return common.TopUpStatusSuccess
	case "failed", "expired":
		return common.TopUpStatusExpired
	default:
		return common.TopUpStatusPending
	}
}
