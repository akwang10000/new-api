package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestRecordLogWithAdminInfoStoresAdminInfoOutsideContent(t *testing.T) {
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))
	t.Cleanup(func() {
		LOG_DB.Exec("DELETE FROM logs")
	})

	RecordLogWithAdminInfo(1, LogTypeManage, "管理员强制禁用了用户的两步验证", map[string]interface{}{
		"admin_id":       2,
		"admin_username": "root-admin",
	})

	var log Log
	require.NoError(t, LOG_DB.Where("user_id = ?", 1).First(&log).Error)
	require.NotContains(t, log.Content, "root-admin")
	require.NotContains(t, log.Content, "ID:2")

	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	require.Contains(t, other, "admin_info")

	formatUserLogs([]*Log{&log}, 0)
	require.NotContains(t, log.Other, "admin_info")
}
