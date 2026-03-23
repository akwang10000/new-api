package controller

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	PaymentMethodBEpusdt       = "bepusdt"
	paymentMethodBEpusdtPrefix = "bepusdt_"
	bepusdtMaxTopup            = 10000
	bepusdtWebhookMaxBody      = 1 << 20
)

type BEpusdtPayRequest struct {
	Amount        int64  `json:"amount"`
	TradeType     string `json:"trade_type"`
	PaymentMethod string `json:"payment_method"`
}

func RequestBEpusdtPay(c *gin.Context) {
	var req BEpusdtPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "invalid request"})
		return
	}
	if req.PaymentMethod != PaymentMethodBEpusdt {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "unsupported payment method"})
		return
	}
	if !service.IsBEpusdtEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BEpusdt is not configured"})
		return
	}
	if strings.TrimSpace(system_setting.ServerAddress) == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "server address is not configured"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("top-up amount must be at least %d", getMinTopup())})
		return
	}
	if req.Amount > bepusdtMaxTopup {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("top-up amount must not exceed %d", bepusdtMaxTopup)})
		return
	}

	network, ok := service.GetBEpusdtNetworkByCode(req.TradeType)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "unsupported USDT network"})
		return
	}

	userID := c.GetInt("id")
	priceAmountUSD, err := getBEpusdtCreditAmountUSD(req.Amount)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	priceAmountCNY, err := getBEpusdtChargeAmountCNY(req.Amount)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	if priceAmountUSD <= 0.01 || priceAmountCNY <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "payment amount is too small"})
		return
	}

	callbackAddress := strings.TrimRight(strings.TrimSpace(service.GetCallbackAddress()), "/")
	redirectAddress := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if callbackAddress == "" || redirectAddress == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "callback address is not configured"})
		return
	}
	notifyURL, err := getBEpusdtNotifyURL(callbackAddress)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	client := service.NewBEpusdtClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BEpusdt is not configured"})
		return
	}

	reference := fmt.Sprintf("bepusdt-ref-%d-%d-%s", userID, time.Now().UnixMilli(), randstr.String(4))
	tradeNo := "ref_" + common.Sha1([]byte(reference))
	topUp := &model.TopUp{
		UserId:        userID,
		Amount:        req.Amount,
		Money:         priceAmountUSD,
		TradeNo:       tradeNo,
		PaymentMethod: getBEpusdtPaymentMethod(network.Code),
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err = topUp.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "failed to create order"})
		return
	}

	requestBody := &service.BEpusdtCreateTransactionRequest{
		OrderID:     tradeNo,
		Amount:      priceAmountCNY,
		NotifyURL:   notifyURL,
		RedirectURL: redirectAddress + "/console/topup",
		TradeType:   network.Code,
		Fiat:        operation_setting.QuotaDisplayTypeCNY,
		Name:        fmt.Sprintf("new-api topup %s", network.Name),
		Timeout:     setting.BEpusdtOrderTimeout,
	}
	requestBody.Signature = service.SignBEpusdtParams(map[string]string{
		"order_id":     tradeNo,
		"amount":       service.CanonicalBEpusdtDecimal(priceAmountCNY, 2),
		"notify_url":   requestBody.NotifyURL,
		"redirect_url": requestBody.RedirectURL,
		"trade_type":   requestBody.TradeType,
		"fiat":         requestBody.Fiat,
		"name":         requestBody.Name,
		"timeout":      fmt.Sprintf("%d", requestBody.Timeout),
	})

	resp, err := client.CreateTransaction(requestBody)
	if err != nil {
		log.Printf("create bepusdt transaction failed: trade_no=%s err=%v", tradeNo, err)
		if expireErr := model.ExpireTopUp(tradeNo); expireErr != nil {
			log.Printf("expire failed bepusdt transaction order failed: trade_no=%s err=%v", tradeNo, expireErr)
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "error",
			"data":    getBEpusdtLaunchErrorMessage(err),
		})
		return
	}
	if resp.Data == nil || strings.TrimSpace(resp.Data.PaymentURL) == "" || strings.TrimSpace(resp.Data.TradeID) == "" {
		log.Printf("bepusdt response is incomplete: trade_no=%s", tradeNo)
		if expireErr := model.ExpireTopUp(tradeNo); expireErr != nil {
			log.Printf("expire incomplete bepusdt order failed: trade_no=%s err=%v", tradeNo, expireErr)
		}
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BEpusdt payment link is unavailable"})
		return
	}
	if strings.TrimSpace(resp.Data.OrderID) != "" && strings.TrimSpace(resp.Data.OrderID) != tradeNo {
		log.Printf("bepusdt response order_id mismatch: trade_no=%s order_id=%s", tradeNo, strings.TrimSpace(resp.Data.OrderID))
		if expireErr := model.ExpireTopUp(tradeNo); expireErr != nil {
			log.Printf("expire mismatched bepusdt order failed: trade_no=%s err=%v", tradeNo, expireErr)
		}
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "BEpusdt order verification failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": resp.Data.PaymentURL,
		},
	})
}

func BEpusdtWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, bepusdtWebhookMaxBody+1))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if len(body) > bepusdtWebhookMaxBody {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}

	params, err := parseBEpusdtCallbackParams(body, c.ContentType())
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	signature := strings.TrimSpace(params["signature"])
	if !service.VerifyBEpusdtSignature(params, signature) {
		log.Printf("bepusdt webhook signature verify failed")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	tradeNo := strings.TrimSpace(params["order_id"])
	if tradeNo == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	tradeID := strings.TrimSpace(params["trade_id"])
	if tradeID == "" {
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

	client := service.NewBEpusdtClient()
	if client == nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	remoteOrder, err := client.QueryOrder(tradeID)
	if err != nil {
		log.Printf("bepusdt query order failed: trade_no=%s trade_id=%s err=%v", tradeNo, tradeID, err)
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}

	if err = validateBEpusdtCallback(topUp, params, remoteOrder); err != nil {
		log.Printf("bepusdt webhook validation failed: trade_no=%s err=%v", tradeNo, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch normalizeBEpusdtRemoteStatus(remoteOrder.Status) {
	case common.TopUpStatusSuccess:
		if topUp.Status == common.TopUpStatusSuccess {
			c.String(http.StatusOK, "success")
			return
		}
		if topUp.Status != common.TopUpStatusPending {
			log.Printf("ignore successful bepusdt webhook for non-pending order: trade_no=%s status=%s", tradeNo, topUp.Status)
			break
		}
		if err = model.CompleteTopUpByMoney(tradeNo, nil); err != nil {
			log.Printf("complete bepusdt topup failed: trade_no=%s err=%v", tradeNo, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("bepusdt topup completed: trade_no=%s trade_id=%s", tradeNo, strings.TrimSpace(params["trade_id"]))
	case common.TopUpStatusExpired:
		if err = model.ExpireTopUp(tradeNo); err != nil {
			log.Printf("expire bepusdt topup failed: trade_no=%s err=%v", tradeNo, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("bepusdt topup expired: trade_no=%s trade_id=%s", tradeNo, strings.TrimSpace(params["trade_id"]))
	default:
		log.Printf("ignore bepusdt webhook: trade_no=%s status=%s", tradeNo, strings.TrimSpace(params["status"]))
	}

	c.String(http.StatusOK, "success")
}

func parseBEpusdtCallbackParams(body []byte, contentType string) (map[string]string, error) {
	params := make(map[string]string)
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "application/json") {
		var payload map[string]any
		if err := common.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		for key, value := range payload {
			if value == nil {
				continue
			}
			params[key] = strings.TrimSpace(fmt.Sprint(value))
		}
		return params, nil
	}

	values, err := parseBEpusdtFormEncoded(string(body))
	if err != nil {
		return nil, err
	}
	for key, value := range values {
		params[key] = strings.TrimSpace(value)
	}
	return params, nil
}

func parseBEpusdtFormEncoded(raw string) (map[string]string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(values))
	for key := range values {
		result[key] = values.Get(key)
	}
	return result, nil
}

func getBEpusdtPaymentMethod(tradeType string) string {
	return paymentMethodBEpusdtPrefix + normalizeBEpusdtTradeType(tradeType)
}

func getBEpusdtTradeTypeFromPaymentMethod(paymentMethod string) string {
	paymentMethod = strings.ToLower(strings.TrimSpace(paymentMethod))
	if !strings.HasPrefix(paymentMethod, paymentMethodBEpusdtPrefix) {
		return ""
	}
	return strings.TrimPrefix(paymentMethod, paymentMethodBEpusdtPrefix)
}

func normalizeBEpusdtTradeType(tradeType string) string {
	tradeType = strings.ToLower(strings.TrimSpace(tradeType))
	replacer := strings.NewReplacer(".", "_", "-", "_", " ", "_")
	return replacer.Replace(tradeType)
}

func validateBEpusdtCallback(topUp *model.TopUp, params map[string]string, remoteOrder *service.BEpusdtQueryOrderResponse) error {
	if topUp == nil {
		return fmt.Errorf("topup is nil")
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(topUp.PaymentMethod)), paymentMethodBEpusdtPrefix) {
		return fmt.Errorf("payment method mismatch")
	}
	if strings.TrimSpace(params["order_id"]) != topUp.TradeNo {
		return fmt.Errorf("trade no mismatch")
	}
	if err := model.ValidateBEpusdtTopUp(topUp); err != nil {
		return err
	}

	expectedAmount, err := getExpectedBEpusdtCallbackAmount(topUp)
	if err != nil {
		return err
	}
	actualAmount, err := decimal.NewFromString(strings.TrimSpace(params["amount"]))
	if err != nil {
		return fmt.Errorf("invalid amount")
	}
	if !actualAmount.Round(2).Equal(expectedAmount) {
		return fmt.Errorf("amount mismatch")
	}
	if remoteOrder == nil {
		return fmt.Errorf("remote order is nil")
	}
	if remoteOrder.Status <= 0 {
		return fmt.Errorf("invalid remote status")
	}
	callbackTradeID := strings.TrimSpace(params["trade_id"])
	if callbackTradeID == "" {
		return fmt.Errorf("missing trade_id")
	}
	if !strings.EqualFold(remoteOrder.TradeID, callbackTradeID) {
		return fmt.Errorf("trade_id mismatch")
	}
	if actualAmount := strings.TrimSpace(params["actual_amount"]); actualAmount == "" {
		return fmt.Errorf("missing actual_amount")
	}
	if token := strings.TrimSpace(params["token"]); token == "" {
		return fmt.Errorf("missing token")
	}
	if tradeType := strings.TrimSpace(params["trade_type"]); tradeType != "" {
		if normalizeBEpusdtTradeType(tradeType) != getBEpusdtTradeTypeFromPaymentMethod(topUp.PaymentMethod) {
			return fmt.Errorf("trade_type mismatch")
		}
	}
	if fiat := strings.TrimSpace(params["fiat"]); fiat != "" && !strings.EqualFold(fiat, operation_setting.QuotaDisplayTypeCNY) {
		return fmt.Errorf("fiat mismatch")
	}
	if callbackStatus := strings.TrimSpace(params["status"]); callbackStatus != "" && normalizeBEpusdtStatus(callbackStatus) != normalizeBEpusdtRemoteStatus(remoteOrder.Status) {
		return fmt.Errorf("status mismatch")
	}
	return nil
}

func normalizeBEpusdtStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "2":
		return common.TopUpStatusSuccess
	case "3":
		return common.TopUpStatusExpired
	default:
		return common.TopUpStatusPending
	}
}

func normalizeBEpusdtRemoteStatus(status int) string {
	switch status {
	case 2:
		return common.TopUpStatusSuccess
	case 3:
		return common.TopUpStatusExpired
	default:
		return common.TopUpStatusPending
	}
}

func getBEpusdtChargeAmountCNY(amount int64) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("payment amount is too small")
	}
	return decimal.NewFromInt(amount).Round(2).InexactFloat64(), nil
}

func getBEpusdtCreditAmountUSD(amount int64) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("payment amount is too small")
	}
	return decimal.NewFromInt(amount).Round(2).InexactFloat64(), nil
}

func getExpectedBEpusdtCallbackAmount(topUp *model.TopUp) (decimal.Decimal, error) {
	if topUp == nil {
		return decimal.Zero, fmt.Errorf("topup is nil")
	}
	if topUp.Amount <= 0 {
		return decimal.Zero, fmt.Errorf("invalid bepusdt order amount")
	}
	return decimal.NewFromInt(topUp.Amount).Round(2), nil
}

func getBEpusdtLaunchErrorMessage(err error) string {
	return "failed to launch BEpusdt payment"
}

func getBEpusdtNotifyURL(callbackAddress string) (string, error) {
	notifyURL, err := url.Parse(strings.TrimRight(callbackAddress, "/") + "/api/bepusdt/webhook")
	if err != nil {
		return "", fmt.Errorf("failed to build BEpusdt webhook URL")
	}
	return notifyURL.String(), nil
}
