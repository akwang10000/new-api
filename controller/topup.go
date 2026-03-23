package controller

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func filterVisiblePayMethods(payMethods []map[string]string, enableOnlineTopUp bool) []map[string]string {
	if len(payMethods) == 0 {
		return payMethods
	}

	filtered := make([]map[string]string, 0, len(payMethods))
	for _, method := range payMethods {
		methodType := strings.TrimSpace(method["type"])
		if methodType == "" {
			continue
		}
		switch methodType {
		case "stripe", PaymentMethodBTCPay, PaymentMethodBEpusdt, PaymentMethodNOWPayments:
			filtered = append(filtered, method)
		default:
			if enableOnlineTopUp {
				filtered = append(filtered, method)
			}
		}
	}
	return filtered
}

func GetTopUpInfo(c *gin.Context) {
	// 获取支付方式
	enableOnlineTopUp := operation_setting.PayAddress != "" && operation_setting.EpayId != "" && operation_setting.EpayKey != ""
	payMethods := filterVisiblePayMethods(operation_setting.PayMethods, enableOnlineTopUp)
	enableStripeTopUp := setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != ""
	enableBTCPayTopUp := service.IsBTCPayEnabled()
	enableBEpusdtTopUp := service.IsBEpusdtEnabled()
	bepusdtNetworks := service.GetEnabledBEpusdtNetworks()
	enableNOWPaymentsTopUp := service.IsNOWPaymentsEnabled()
	nowPaymentsNetworks := service.GetEnabledNOWPaymentsNetworks()
	nowPaymentsCryptoAmountOptions, _ := service.ParseNOWPaymentsCryptoAmountOptions(setting.NOWPaymentsCryptoAmountOptions)

	// 如果启用了 Stripe 支付，添加到支付方法列表
	if enableStripeTopUp {
		// 检查是否已经包含 Stripe
		hasStripe := false
		for _, method := range payMethods {
			if method["type"] == "stripe" {
				hasStripe = true
				break
			}
		}

		if !hasStripe {
			stripeMethod := map[string]string{
				"name":      "Stripe",
				"type":      "stripe",
				"color":     "rgba(var(--semi-purple-5), 1)",
				"min_topup": strconv.Itoa(setting.StripeMinTopUp),
			}
			payMethods = append(payMethods, stripeMethod)
		}
	}
	if enableBTCPayTopUp {
		hasBTCPay := false
		for _, method := range payMethods {
			if method["type"] == PaymentMethodBTCPay {
				hasBTCPay = true
				break
			}
		}

		if !hasBTCPay {
			payMethods = append(payMethods, map[string]string{
				"name":      "BTCPay",
				"type":      PaymentMethodBTCPay,
				"color":     "rgba(var(--semi-orange-5), 1)",
				"min_topup": strconv.Itoa(operation_setting.MinTopUp),
			})
		}
	}
	if enableNOWPaymentsTopUp {
		hasNOWPayments := false
		for _, method := range payMethods {
			if method["type"] == PaymentMethodNOWPayments {
				hasNOWPayments = true
				break
			}
		}

		if !hasNOWPayments {
			payMethods = append(payMethods, map[string]string{
				"name":      "NOWPayments",
				"type":      PaymentMethodNOWPayments,
				"color":     "rgba(var(--semi-teal-5), 1)",
				"min_topup": strconv.Itoa(operation_setting.MinTopUp),
			})
		}
	}
	if enableBEpusdtTopUp {
		hasBEpusdt := false
		for _, method := range payMethods {
			if method["type"] == PaymentMethodBEpusdt {
				hasBEpusdt = true
				break
			}
		}

		if !hasBEpusdt {
			payMethods = append(payMethods, map[string]string{
				"name":      "虚拟货币支付",
				"type":      PaymentMethodBEpusdt,
				"color":     "rgba(var(--semi-cyan-5), 1)",
				"min_topup": strconv.Itoa(operation_setting.MinTopUp),
			})
		}
	}

	data := gin.H{
		"enable_online_topup":               enableOnlineTopUp,
		"enable_stripe_topup":               enableStripeTopUp,
		"enable_creem_topup":                setting.CreemApiKey != "" && setting.CreemProducts != "[]",
		"enable_btcpay_topup":               enableBTCPayTopUp,
		"enable_bepusdt_topup":              enableBEpusdtTopUp,
		"bepusdt_usdt_networks":             bepusdtNetworks,
		"enable_nowpayments_topup":          enableNOWPaymentsTopUp,
		"nowpayments_modes":                 gin.H{"fiat": setting.NOWPaymentsFiatModeEnabled, "crypto": setting.NOWPaymentsCryptoModeEnabled},
		"nowpayments_usdt_networks":         nowPaymentsNetworks,
		"nowpayments_crypto_amount_options": nowPaymentsCryptoAmountOptions,
		"creem_products":                    setting.CreemProducts,
		"pay_methods":                       payMethods,
		"min_topup":                         operation_setting.MinTopUp,
		"stripe_min_topup":                  setting.StripeMinTopUp,
		"amount_options":                    operation_setting.GetPaymentSetting().AmountOptions,
		"discount":                          operation_setting.GetPaymentSetting().AmountDiscount,
	}
	common.ApiSuccess(c, data)
}

type EpayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

type AmountRequest struct {
	Amount int64 `json:"amount"`
}

func GetEpayClient() *epay.Client {
	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		return nil
	}
	withUrl, err := epay.NewClient(&epay.Config{
		PartnerID: operation_setting.EpayId,
		Key:       operation_setting.EpayKey,
	}, operation_setting.PayAddress)
	if err != nil {
		return nil
	}
	return withUrl
}

func getPayMoney(amount int64, group string) float64 {
	dAmount := decimal.NewFromInt(amount)
	// 充值金额以“展示类型”为准：
	// - USD/CNY: 前端传 amount 为金额单位；TOKENS: 前端传 tokens，需要换成 USD 金额
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dAmount = dAmount.Div(dQuotaPerUnit)
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	dTopupGroupRatio := decimal.NewFromFloat(topupGroupRatio)
	dPrice := decimal.NewFromFloat(operation_setting.Price)
	// apply optional preset discount by the original request amount (if configured), default 1.0
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	dDiscount := decimal.NewFromFloat(discount)

	payMoney := dAmount.Mul(dPrice).Mul(dTopupGroupRatio).Mul(dDiscount)

	return payMoney.InexactFloat64()
}

func getPayMoneyUSD(amount int64, group string) (float64, error) {
	rate := decimal.NewFromFloat(operation_setting.USDExchangeRate)
	if rate.LessThanOrEqual(decimal.Zero) {
		return 0, fmt.Errorf("USD 姹囩巼閰嶇疆閿欒")
	}
	return decimal.NewFromFloat(getPayMoney(amount, group)).Div(rate).Round(2).InexactFloat64(), nil
}

func getMinTopup() int64 {
	minTopup := operation_setting.MinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dMinTopup := decimal.NewFromInt(int64(minTopup))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		minTopup = int(dMinTopup.Mul(dQuotaPerUnit).IntPart())
	}
	return int64(minTopup)
}

func getPayMoneyCNY(amount int64, group string) (float64, error) {
	payMoney := decimal.NewFromFloat(getPayMoney(amount, group))
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		return payMoney.Round(2).InexactFloat64(), nil
	case operation_setting.QuotaDisplayTypeUSD, operation_setting.QuotaDisplayTypeTokens:
		rate := decimal.NewFromFloat(operation_setting.USDExchangeRate)
		if rate.LessThanOrEqual(decimal.Zero) {
			return 0, fmt.Errorf("USD exchange rate is invalid")
		}
		return payMoney.Mul(rate).Round(2).InexactFloat64(), nil
	case operation_setting.QuotaDisplayTypeCustom:
		customRate := decimal.NewFromFloat(operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate)
		if customRate.LessThanOrEqual(decimal.Zero) {
			return 0, fmt.Errorf("custom currency exchange rate is invalid")
		}
		usdRate := decimal.NewFromFloat(operation_setting.USDExchangeRate)
		if usdRate.LessThanOrEqual(decimal.Zero) {
			return 0, fmt.Errorf("USD exchange rate is invalid")
		}
		return payMoney.Div(customRate).Mul(usdRate).Round(2).InexactFloat64(), nil
	default:
		rate := decimal.NewFromFloat(operation_setting.USDExchangeRate)
		if rate.LessThanOrEqual(decimal.Zero) {
			return 0, fmt.Errorf("USD exchange rate is invalid")
		}
		return payMoney.Mul(rate).Round(2).InexactFloat64(), nil
	}
}

func RequestEpay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		c.JSON(200, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl, _ := url.Parse(system_setting.ServerAddress + "/console/log")
	notifyUrl, _ := url.Parse(callBackAddress + "/api/user/epay/notify")
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)
	client := GetEpayClient()
	if client == nil {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:        id,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: req.PaymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        "pending",
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	checkout, err := CreateEpayCheckout(c, &epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("TUC%d", req.Amount),
		Money:          strconv.FormatFloat(payMoney, 'f', 2, 64),
		Device:         epay.PC,
		NotifyUrl:      notifyUrl,
		ReturnUrl:      returnUrl,
	})
	if err != nil {
		if expireErr := model.ExpireTopUp(tradeNo); expireErr != nil {
			log.Printf("expire failed epay order failed: trade_no=%s err=%v", tradeNo, expireErr)
		}
		c.JSON(200, gin.H{"message": "error", "data": err.Error()})
		return
	}
	if checkout.PayLink != "" {
		c.JSON(200, gin.H{
			"message": "success",
			"data": buildEpayCheckoutPayload(checkout, gin.H{
				"trade_no":             tradeNo,
				"subject":              "钱包充值",
				"payment_method":       req.PaymentMethod,
				"payment_method_label": getEpayMethodDisplayName(req.PaymentMethod),
				"pay_amount":           strconv.FormatFloat(payMoney, 'f', 2, 64),
				"pay_currency":         "CNY",
				"recharge_amount":      req.Amount,
			}),
		})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": checkout.Params, "url": checkout.URL})
}

// tradeNo lock
var orderLocks sync.Map
var createLock sync.Mutex

// LockOrder 尝试对给定订单号加锁
func LockOrder(tradeNo string) {
	lock, ok := orderLocks.Load(tradeNo)
	if !ok {
		createLock.Lock()
		defer createLock.Unlock()
		lock, ok = orderLocks.Load(tradeNo)
		if !ok {
			lock = new(sync.Mutex)
			orderLocks.Store(tradeNo, lock)
		}
	}
	lock.(*sync.Mutex).Lock()
}

// UnlockOrder 释放给定订单号的锁
func UnlockOrder(tradeNo string) {
	lock, ok := orderLocks.Load(tradeNo)
	if ok {
		lock.(*sync.Mutex).Unlock()
	}
}

func EpayNotify(c *gin.Context) {
	var params map[string]string

	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err != nil {
			log.Println("epay notify parse form failed:", err)
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, _ int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, _ int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		log.Println("epay notify params are empty")
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	epayClient := GetEpayClient()
	if epayClient == nil {
		log.Println("epay notify missing payment config")
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	verifiedInfo, err := epayClient.Verify(params)
	if err != nil || verifiedInfo == nil || !verifiedInfo.VerifyStatus {
		log.Println("epay notify signature verification failed")
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if verifiedInfo.TradeStatus != epay.StatusTradeSuccess {
		log.Printf("ignore epay notify with non-success trade status: %+v", verifiedInfo)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(verifiedInfo.ServiceTradeNo)
	defer UnlockOrder(verifiedInfo.ServiceTradeNo)

	existingTopUp := model.GetTopUpByTradeNo(verifiedInfo.ServiceTradeNo)
	if existingTopUp == nil {
		log.Printf("epay notify top-up not found: %+v", verifiedInfo)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if existingTopUp.Status == common.TopUpStatusSuccess {
		_, _ = c.Writer.Write([]byte("success"))
		return
	}
	if existingTopUp.Status != common.TopUpStatusPending {
		log.Printf("epay notify ignore non-pending order: trade_no=%s status=%s", verifiedInfo.ServiceTradeNo, existingTopUp.Status)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if strings.TrimSpace(verifiedInfo.Type) != "" && !strings.EqualFold(strings.TrimSpace(verifiedInfo.Type), strings.TrimSpace(existingTopUp.PaymentMethod)) {
		log.Printf("epay notify payment method mismatch: trade_no=%s expected=%s actual=%s", verifiedInfo.ServiceTradeNo, existingTopUp.PaymentMethod, verifiedInfo.Type)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	paidMoney, err := decimal.NewFromString(strings.TrimSpace(verifiedInfo.Money))
	if err != nil {
		log.Printf("epay notify invalid money: trade_no=%s money=%q err=%v", verifiedInfo.ServiceTradeNo, verifiedInfo.Money, err)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	expectedMoney := decimal.NewFromFloat(existingTopUp.Money).Round(2)
	if !paidMoney.Round(2).Equal(expectedMoney) {
		log.Printf("epay notify money mismatch: trade_no=%s expected=%s actual=%s", verifiedInfo.ServiceTradeNo, expectedMoney.StringFixed(2), paidMoney.Round(2).StringFixed(2))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	if err = model.CompleteTopUpByRequestedAmount(verifiedInfo.ServiceTradeNo); err != nil {
		log.Printf("epay notify complete top-up failed: trade_no=%s err=%v", verifiedInfo.ServiceTradeNo, err)
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	log.Printf("epay top-up completed: trade_no=%s gateway_trade_no=%s", verifiedInfo.ServiceTradeNo, verifiedInfo.TradeNo)
	_, _ = c.Writer.Write([]byte("success"))
	return
}

/*

	if c.Request.Method == "POST" {
		// POST 请求：从 POST body 解析参数
		if err := c.Request.ParseForm(); err != nil {
			log.Println("易支付回调POST解析失败:", err)
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		// GET 请求：从 URL Query 解析参数
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}

	if len(params) == 0 {
		log.Println("易支付回调参数为空")
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	client := GetEpayClient()
	if client == nil {
		log.Println("易支付回调失败 未找到配置信息")
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
		return
	}
	verifyInfo, err := client.Verify(params)
	if err == nil && verifyInfo.VerifyStatus {
		_, err := c.Writer.Write([]byte("success"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
	} else {
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			log.Println("易支付回调写入失败")
		}
		log.Println("易支付回调签名验证失败")
		return
	}

	if verifyInfo.TradeStatus == epay.StatusTradeSuccess {
		log.Println(verifyInfo)
		LockOrder(verifyInfo.ServiceTradeNo)
		defer UnlockOrder(verifyInfo.ServiceTradeNo)
		topUp := model.GetTopUpByTradeNo(verifyInfo.ServiceTradeNo)
		if topUp == nil {
			log.Printf("易支付回调未找到订单: %v", verifyInfo)
			return
		}
		if topUp.Status == "pending" {
			topUp.Status = "success"
			err := topUp.Update()
			if err != nil {
				log.Printf("易支付回调更新订单失败: %v", topUp)
				return
			}
			//user, _ := model.GetUserById(topUp.UserId, false)
			//user.Quota += topUp.Amount * 500000
			dAmount := decimal.NewFromInt(int64(topUp.Amount))
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
			err = model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true)
			if err != nil {
				log.Printf("易支付回调更新用户失败: %v", topUp)
				return
			}
			log.Printf("易支付回调更新用户成功 %v", topUp)
			model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money))
		}
	} else {
		log.Printf("易支付异常回调: %v", verifyInfo)
	}
}
*/

func RequestAmount(c *gin.Context) {
	var req AmountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func GetUserTopUps(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchUserTopUps(userId, keyword, pageInfo)
	} else {
		topups, total, err = model.GetUserTopUps(userId, pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

// GetAllTopUps 管理员获取全平台充值记录
func GetAllTopUps(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	var (
		topups []*model.TopUp
		total  int64
		err    error
	)
	if keyword != "" {
		topups, total, err = model.SearchAllTopUps(keyword, pageInfo)
	} else {
		topups, total, err = model.GetAllTopUps(pageInfo)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

type AdminCompleteTopupRequest struct {
	TradeNo string `json:"trade_no"`
}

// AdminCompleteTopUp 管理员补单接口
func AdminCompleteTopUp(c *gin.Context) {
	var req AdminCompleteTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	// 订单级互斥，防止并发补单
	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	if err := model.ManualCompleteTopUp(req.TradeNo); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
