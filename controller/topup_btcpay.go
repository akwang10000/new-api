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
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

const (
	PaymentMethodBTCPay   = "btcpay"
	BTCPaySignatureHeader = "BTCPay-Sig"
	btcpayWebhookMaxBody  = 1 << 20
	btcpayMaxTopup        = 10000
)

type BTCPayPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func RequestBTCPayPay(c *gin.Context) {
	var req BTCPayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.PaymentMethod != PaymentMethodBTCPay {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "不支持的支付渠道"})
		return
	}
	if !service.IsBTCPayEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BTCPay 未配置或未启用"})
		return
	}
	if strings.TrimSpace(system_setting.ServerAddress) == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "请先配置服务器地址"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}
	if req.Amount > btcpayMaxTopup {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能大于 %d", btcpayMaxTopup)})
		return
	}

	userID := c.GetInt("id")
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	invoiceAmount, err := getBTCPayInvoiceAmountUSD(payMoney)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	if invoiceAmount < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BTCPay 支付金额过低"})
		return
	}

	client := service.NewBTCPayClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BTCPay 未配置或未启用"})
		return
	}

	reference := fmt.Sprintf("btcpay-ref-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(4))
	referenceID := "ref_" + common.Sha1([]byte(reference))

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}

	topUp := &model.TopUp{
		UserId:        userID,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       referenceID,
		PaymentMethod: PaymentMethodBTCPay,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err = topUp.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	redirectURL := strings.TrimRight(system_setting.ServerAddress, "/") + "/console/topup"
	invoice, err := client.CreateInvoice(&service.BTCPayCreateInvoiceRequest{
		Amount:   invoiceAmount,
		Currency: "USD",
		Metadata: map[string]any{
			"orderId":       referenceID,
			"tradeNo":       referenceID,
			"userId":        userID,
			"topupAmount":   amount,
			"paymentMethod": PaymentMethodBTCPay,
			"payMoney":      payMoney,
		},
		Checkout: &service.BTCPayInvoiceCheckout{
			RedirectURL:           redirectURL,
			RedirectAutomatically: true,
		},
	})
	if err != nil {
		log.Printf("create btcpay invoice failed: trade_no=%s err=%v", referenceID, err)
		if expireErr := model.ExpireTopUp(referenceID); expireErr != nil {
			log.Printf("expire failed btcpay order failed: trade_no=%s err=%v", referenceID, expireErr)
		}
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起 BTCPay 支付失败"})
		return
	}
	if strings.TrimSpace(invoice.CheckoutLink) == "" {
		log.Printf("btcpay invoice missing checkout link: trade_no=%s invoice_id=%s", referenceID, invoice.ID)
		if expireErr := model.ExpireTopUp(referenceID); expireErr != nil {
			log.Printf("expire missing-link btcpay order failed: trade_no=%s err=%v", referenceID, expireErr)
		}
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BTCPay 支付链接不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": invoice.CheckoutLink,
		},
	})
}

func BTCPayWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, btcpayWebhookMaxBody+1))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if len(body) > btcpayWebhookMaxBody {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}

	if !service.VerifyBTCPaySignature(body, c.GetHeader(BTCPaySignatureHeader)) {
		log.Printf("btcpay webhook signature verify failed")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var event service.BTCPayWebhookEvent
	if err = common.Unmarshal(body, &event); err != nil {
		log.Printf("decode btcpay webhook failed: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(event.InvoiceID) == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if event.StoreID != "" && strings.TrimSpace(event.StoreID) != strings.TrimSpace(setting.BTCPayStoreID) {
		log.Printf("btcpay webhook store id mismatch: invoice_id=%s store_id=%s", event.InvoiceID, event.StoreID)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	client := service.NewBTCPayClient()
	if client == nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	invoice, err := client.GetInvoice(event.InvoiceID)
	if err != nil {
		log.Printf("fetch btcpay invoice failed: invoice_id=%s err=%v", event.InvoiceID, err)
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	tradeNo := getBTCPayTradeNo(invoice.Metadata)
	if tradeNo == "" {
		log.Printf("btcpay webhook missing trade_no: invoice_id=%s", event.InvoiceID)
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
	if err = validateBTCPayInvoice(topUp, invoice); err != nil {
		log.Printf("btcpay webhook invoice validation failed: trade_no=%s invoice_id=%s err=%v", tradeNo, event.InvoiceID, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch normalizeBTCPayInvoiceState(invoice.Status, event.Type) {
	case common.TopUpStatusSuccess:
		if topUp.Status == common.TopUpStatusSuccess {
			c.Status(http.StatusOK)
			return
		}
		if topUp.Status != common.TopUpStatusPending {
			log.Printf("ignore settled btcpay webhook for non-pending order: trade_no=%s status=%s", tradeNo, topUp.Status)
			break
		}
		if err = model.CompleteTopUpByMoney(tradeNo, nil, c.ClientIP(), "btcpay"); err != nil {
			log.Printf("complete btcpay topup failed: trade_no=%s err=%v", tradeNo, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("btcpay topup completed: trade_no=%s invoice_id=%s", tradeNo, event.InvoiceID)
	case common.TopUpStatusExpired:
		if err = model.ExpireTopUp(tradeNo); err != nil {
			log.Printf("expire btcpay topup failed: trade_no=%s err=%v", tradeNo, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("btcpay topup expired: trade_no=%s invoice_id=%s", tradeNo, event.InvoiceID)
	default:
		log.Printf("ignore btcpay webhook: trade_no=%s invoice_id=%s type=%s status=%s", tradeNo, event.InvoiceID, event.Type, invoice.Status)
	}

	c.Status(http.StatusOK)
}

func getBTCPayInvoiceAmountUSD(payMoney float64) (float64, error) {
	rate := decimal.NewFromFloat(operation_setting.USDExchangeRate)
	if rate.LessThanOrEqual(decimal.Zero) {
		return 0, fmt.Errorf("USD 汇率配置错误")
	}
	usdAmount := decimal.NewFromFloat(payMoney).Div(rate).Round(2)
	return usdAmount.InexactFloat64(), nil
}

func getBTCPayTradeNo(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	for _, key := range []string{"orderId", "tradeNo", "referenceId"} {
		value := strings.TrimSpace(fmt.Sprint(metadata[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func validateBTCPayInvoice(topUp *model.TopUp, invoice *service.BTCPayInvoiceResponse) error {
	if topUp == nil {
		return fmt.Errorf("topup is nil")
	}
	if invoice == nil {
		return fmt.Errorf("invoice is nil")
	}
	if topUp.PaymentMethod != PaymentMethodBTCPay {
		return fmt.Errorf("payment method mismatch: %s", topUp.PaymentMethod)
	}
	if getBTCPayTradeNo(invoice.Metadata) != topUp.TradeNo {
		return fmt.Errorf("trade no mismatch")
	}
	if strings.ToLower(strings.TrimSpace(fmt.Sprint(invoice.Metadata["paymentMethod"]))) != PaymentMethodBTCPay {
		return fmt.Errorf("metadata payment method mismatch")
	}

	userID, ok := getBTCPayMetadataInt64(invoice.Metadata, "userId")
	if !ok || int(userID) != topUp.UserId {
		return fmt.Errorf("metadata user id mismatch")
	}
	topupAmount, ok := getBTCPayMetadataInt64(invoice.Metadata, "topupAmount")
	if !ok || topupAmount != topUp.Amount {
		return fmt.Errorf("metadata amount mismatch")
	}
	payMoney, ok := getBTCPayMetadataDecimal(invoice.Metadata, "payMoney")
	if !ok || !payMoney.Equal(decimal.NewFromFloat(topUp.Money).Round(8)) {
		return fmt.Errorf("metadata pay money mismatch")
	}

	if strings.ToUpper(strings.TrimSpace(invoice.Currency)) != "USD" {
		return fmt.Errorf("invoice currency mismatch")
	}
	expectedInvoiceAmount, err := getBTCPayInvoiceAmountUSD(topUp.Money)
	if err != nil {
		return err
	}
	invoiceAmount, err := getBTCPayInvoiceAmount(invoice.Amount)
	if err != nil {
		return err
	}
	if !invoiceAmount.Round(2).Equal(decimal.NewFromFloat(expectedInvoiceAmount).Round(2)) {
		return fmt.Errorf("invoice amount mismatch")
	}
	return nil
}

func getBTCPayMetadataInt64(metadata map[string]any, key string) (int64, bool) {
	if len(metadata) == 0 {
		return 0, false
	}
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return decimal.NewFromFloat(typed).IntPart(), true
	case string:
		if strings.TrimSpace(typed) == "" {
			return 0, false
		}
		number, err := decimal.NewFromString(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return number.IntPart(), true
	default:
		number, err := decimal.NewFromString(strings.TrimSpace(fmt.Sprint(value)))
		if err != nil {
			return 0, false
		}
		return number.IntPart(), true
	}
}

func getBTCPayMetadataDecimal(metadata map[string]any, key string) (decimal.Decimal, bool) {
	if len(metadata) == 0 {
		return decimal.Zero, false
	}
	value, ok := metadata[key]
	if !ok {
		return decimal.Zero, false
	}
	decimalValue, err := getBTCPayInvoiceAmount(value)
	if err != nil {
		return decimal.Zero, false
	}
	return decimalValue.Round(8), true
}

func getBTCPayInvoiceAmount(value any) (decimal.Decimal, error) {
	switch typed := value.(type) {
	case nil:
		return decimal.Zero, fmt.Errorf("invoice amount missing")
	case float64:
		return decimal.NewFromFloat(typed), nil
	case float32:
		return decimal.NewFromFloat(float64(typed)), nil
	case int:
		return decimal.NewFromInt(int64(typed)), nil
	case int64:
		return decimal.NewFromInt(typed), nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return decimal.Zero, fmt.Errorf("invoice amount missing")
		}
		return decimal.NewFromString(strings.TrimSpace(typed))
	default:
		return decimal.NewFromString(strings.TrimSpace(fmt.Sprint(value)))
	}
}

func normalizeBTCPayInvoiceState(status string, eventType string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "settled":
		return common.TopUpStatusSuccess
	case "expired", "invalid":
		return common.TopUpStatusExpired
	}

	eventType = strings.ToLower(strings.TrimSpace(eventType))
	switch {
	case strings.Contains(eventType, "settled"):
		return common.TopUpStatusSuccess
	case strings.Contains(eventType, "expired"), strings.Contains(eventType, "invalid"):
		return common.TopUpStatusExpired
	default:
		return common.TopUpStatusPending
	}
}
