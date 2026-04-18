package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id            int     `json:"id"`
	UserId        int     `json:"user_id" gorm:"index"`
	Amount        int64   `json:"amount"`
	Money         float64 `json:"money"`
	TradeNo       string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod string  `json:"payment_method" gorm:"type:varchar(50)"`
	CreateTime    int64   `json:"create_time"`
	CompleteTime  int64   `json:"complete_time"`
	Status        string  `json:"status"`
}

var ErrPaymentMethodMismatch = errors.New("payment method mismatch")

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func CompleteTopUpByMoney(referenceId string, extraUserUpdates map[string]interface{}) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota float64
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}
		if err := validateMoneyBasedTopUpInvariant(topUp); err != nil {
			return err
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = topUp.Money * common.QuotaPerUnit
		updateFields := map[string]interface{}{
			"quota": gorm.Expr("quota + ?", quota),
		}
		for key, value := range extraUserUpdates {
			if key == "" {
				continue
			}
			updateFields[key] = value
		}
		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(updateFields).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(int(quota)), topUp.Amount))

	return nil
}

func Recharge(referenceId string, customerId string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	topUp := GetTopUpByTradeNo(referenceId)
	if topUp == nil {
		return errors.New("充值订单不存在")
	}
	if !strings.EqualFold(strings.TrimSpace(topUp.PaymentMethod), "stripe") {
		return errors.New("充值订单支付方式错误")
	}

	updateFields := map[string]interface{}{}
	if customerId != "" {
		updateFields["stripe_customer"] = customerId
	}
	return CompleteTopUpByMoney(referenceId, updateFields)
}

func CompleteTopUpByRequestedAmount(referenceId string) error {
	if referenceId == "" {
		return errors.New("missing payment reference id")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	topUp := &TopUp{}
	quotaToAdd := 0

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error; err != nil {
			return errors.New("top-up order does not exist")
		}

		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return errors.New("top-up order is not pending")
		}
		if topUp.Amount <= 0 {
			return errors.New("invalid top-up amount")
		}

		quotaToAdd = int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("invalid top-up quota")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		common.SysError("topup by amount failed: " + err.Error())
		return errors.New("top-up failed, please retry later")
	}

	RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("online top-up succeeded, quota added: %s, payment amount: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money))
	return nil
}

const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

const searchTopUpCountHardLimit = 10000

func countTopUpsWithHardLimit(tx *gorm.DB, query *gorm.DB) (int64, error) {
	var total int64
	subQuery := query.Session(&gorm.Session{}).Select("id").Limit(searchTopUpCountHardLimit)
	err := tx.Table("(?) AS limited_topups", subQuery).Count(&total).Error
	return total, err
}

func topUpSearchLikePattern(keyword string) (string, error) {
	pattern, err := sanitizeLikePattern(strings.TrimSpace(keyword))
	if err != nil {
		return "", err
	}
	if !strings.Contains(pattern, "%") {
		pattern = "%" + pattern + "%"
	}
	return pattern, nil
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		like, patternErr := topUpSearchLikePattern(keyword)
		if patternErr != nil {
			tx.Rollback()
			return nil, 0, patternErr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", like)
	}

	if total, err = countTopUpsWithHardLimit(tx, query); err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号搜索全平台充值记录（管理员使用）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		like, patternErr := topUpSearchLikePattern(keyword)
		if patternErr != nil {
			tx.Rollback()
			return nil, 0, patternErr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", like)
	}

	if total, err = countTopUpsWithHardLimit(tx, query); err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string) error {
	if tradeNo == "" {
		return errors.New("未提供订单号")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	var userId int
	var quotaToAdd int
	var payMoney float64

	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		// 行级锁，避免并发补单
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}

		// 幂等处理：已成功直接返回
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("订单状态不是待支付，无法补单")
		}
		if !isManualTopUpPaymentMethod(topUp.PaymentMethod) {
			return errors.New("this payment order must be completed by the payment gateway callback")
		}
		if err := validateMoneyBasedTopUpInvariant(topUp); err != nil {
			return err
		}

		// 计算应充值额度：
		// - Stripe 订单：Money 代表经分组倍率换算后的美元数量，直接 * QuotaPerUnit
		// - 其他订单（如易支付）：Amount 为美元数量，* QuotaPerUnit
		if isMoneyBasedTopUpPaymentMethod(topUp.PaymentMethod) {
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd = int(decimal.NewFromFloat(topUp.Money).Mul(dQuotaPerUnit).IntPart())
		} else {
			dAmount := decimal.NewFromInt(topUp.Amount)
			dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
			quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		}
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		// 标记完成
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}

		// 增加用户额度（立即写库，保持一致性）
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}

		userId = topUp.UserId
		payMoney = topUp.Money
		return nil
	})

	if err != nil {
		return err
	}

	// 事务外记录日志，避免阻塞
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(quotaToAdd), payMoney))
	return nil
}

func isMoneyBasedTopUpPaymentMethod(paymentMethod string) bool {
	paymentMethod = strings.ToLower(strings.TrimSpace(paymentMethod))
	return paymentMethod == "stripe" || paymentMethod == "btcpay" || strings.HasPrefix(paymentMethod, "nowpayments_") || strings.HasPrefix(paymentMethod, "bepusdt_")
}

func isManualTopUpPaymentMethod(paymentMethod string) bool {
	switch strings.ToLower(strings.TrimSpace(paymentMethod)) {
	case "alipay", "wxpay", "qqpay":
		return true
	default:
		return false
	}
}

func isBEpusdtPaymentMethod(paymentMethod string) bool {
	paymentMethod = strings.ToLower(strings.TrimSpace(paymentMethod))
	return strings.HasPrefix(paymentMethod, "bepusdt_")
}

func ValidateBEpusdtTopUp(topUp *TopUp) error {
	if topUp == nil {
		return errors.New("充值订单不存在")
	}
	if !isBEpusdtPaymentMethod(topUp.PaymentMethod) {
		return errors.New("不是 BEpusdt 订单")
	}
	if topUp.Amount <= 0 {
		return errors.New("无效的 BEpusdt 充值金额")
	}
	if topUp.Money <= 0 {
		return errors.New("无效的 BEpusdt 到账金额")
	}
	expected := decimal.NewFromInt(topUp.Amount).Round(2)
	actual := decimal.NewFromFloat(topUp.Money).Round(2)
	if !actual.Equal(expected) {
		return errors.New("BEpusdt 订单金额校验失败")
	}
	return nil
}

func validateMoneyBasedTopUpInvariant(topUp *TopUp) error {
	if topUp == nil {
		return errors.New("充值订单不存在")
	}
	if isBEpusdtPaymentMethod(topUp.PaymentMethod) {
		return ValidateBEpusdtTopUp(topUp)
	}
	return nil
}

func ExpireTopUp(tradeNo string) error {
	if tradeNo == "" {
		return errors.New("鏈彁渚涜鍗曞彿")
	}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return err
		}
		if topUp.Status != common.TopUpStatusPending {
			return nil
		}
		topUp.Status = common.TopUpStatusExpired
		topUp.CompleteTime = common.GetTimestamp()
		return tx.Save(topUp).Error
	})
}

func RechargeCreem(referenceId string, customerEmail string, customerName string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota int64
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if !strings.EqualFold(strings.TrimSpace(topUp.PaymentMethod), "creem") {
			return ErrPaymentMethodMismatch
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		// Creem 直接使用 Amount 作为充值额度（整数）
		quota = topUp.Amount

		// 构建更新字段，优先使用邮箱，如果邮箱为空则使用用户名
		updateFields := map[string]interface{}{
			"quota": gorm.Expr("quota + ?", quota),
		}

		// 如果有客户邮箱，尝试更新用户邮箱（仅当用户邮箱为空时）
		if customerEmail != "" {
			// 先检查用户当前邮箱是否为空
			var user User
			err = tx.Where("id = ?", topUp.UserId).First(&user).Error
			if err != nil {
				return err
			}

			// 如果用户邮箱为空，则更新为支付时使用的邮箱
			if user.Email == "" {
				updateFields["email"] = customerEmail
			}
		}

		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(updateFields).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}

	RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("使用Creem充值成功，充值额度: %v，支付金额：%.2f", quota, topUp.Money))

	return nil
}
