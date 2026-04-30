package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRelayErrorLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)

	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldErrorLogEnabled := constant.ErrorLogEnabled

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	constant.ErrorLogEnabled = true

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Log{}))

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		constant.ErrorLogEnabled = oldErrorLogEnabled
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestProcessChannelErrorRecordsStreamStatusFromContext(t *testing.T) {
	db := setupRelayErrorLogTestDB(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("id", 12)
	ctx.Set("username", "stream-user")
	ctx.Set("token_name", "stream-token")
	ctx.Set("original_model", "gpt-test")
	ctx.Set("token_id", 34)
	ctx.Set("group", "default")
	ctx.Set("channel_id", 56)
	ctx.Set("channel_name", "test-channel")
	ctx.Set("channel_type", 1)
	ctx.Set(string(constant.ContextKeyRequestStartTime), time.Now())
	ctx.Set(string(constant.ContextKeyIsStream), true)

	processChannelError(ctx, *types.NewChannelError(56, 1, "test-channel", false, "", false), types.NewErrorWithStatusCode(errors.New("upstream failed"), types.ErrorCodeDoRequestFailed, http.StatusBadGateway))

	var log model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeError).First(&log).Error)
	require.True(t, log.IsStream)
}
