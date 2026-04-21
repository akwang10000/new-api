package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type invitationAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type fakeSession struct {
	values map[interface{}]interface{}
}

func newFakeSession() *fakeSession {
	return &fakeSession{values: make(map[interface{}]interface{})}
}

func (s *fakeSession) ID() string { return "test-session" }

func (s *fakeSession) Get(key interface{}) interface{} { return s.values[key] }

func (s *fakeSession) Set(key interface{}, val interface{}) { s.values[key] = val }

func (s *fakeSession) Delete(key interface{}) { delete(s.values, key) }

func (s *fakeSession) Clear() {
	for key := range s.values {
		delete(s.values, key)
	}
}

func (s *fakeSession) AddFlash(value interface{}, vars ...string) {}

func (s *fakeSession) Flashes(vars ...string) []interface{} { return nil }

func (s *fakeSession) Options(options sessions.Options) {}

func (s *fakeSession) Save() error { return nil }

type fakeOAuthProvider struct{}

func (p *fakeOAuthProvider) GetName() string { return "FakeOAuth" }

func (p *fakeOAuthProvider) IsEnabled() bool { return true }

func (p *fakeOAuthProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*oauth.OAuthToken, error) {
	return nil, nil
}

func (p *fakeOAuthProvider) GetUserInfo(ctx context.Context, token *oauth.OAuthToken) (*oauth.OAuthUser, error) {
	return nil, nil
}

func (p *fakeOAuthProvider) IsUserIDTaken(providerUserID string) bool { return false }

func (p *fakeOAuthProvider) FillUserByProviderID(user *model.User, providerUserID string) error { return nil }

func (p *fakeOAuthProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.GitHubId = providerUserID
}

func (p *fakeOAuthProvider) GetProviderPrefix() string { return "oauth_test_" }

func setupInvitationControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)

	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled
	oldRegisterEnabled := common.RegisterEnabled
	oldPasswordRegisterEnabled := common.PasswordRegisterEnabled
	oldEmailVerificationEnabled := common.EmailVerificationEnabled
	oldTurnstileCheckEnabled := common.TurnstileCheckEnabled
	oldQuotaForNewUser := common.QuotaForNewUser
	oldQuotaForInviter := common.QuotaForInviter
	oldQuotaForInvitee := common.QuotaForInvitee
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldGenerateDefaultToken := constant.GenerateDefaultToken

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = false
	common.TurnstileCheckEnabled = false
	common.QuotaForNewUser = 0
	common.QuotaForInviter = 0
	common.QuotaForInvitee = 0
	constant.GenerateDefaultToken = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		common.RegisterEnabled = oldRegisterEnabled
		common.PasswordRegisterEnabled = oldPasswordRegisterEnabled
		common.EmailVerificationEnabled = oldEmailVerificationEnabled
		common.TurnstileCheckEnabled = oldTurnstileCheckEnabled
		common.QuotaForNewUser = oldQuotaForNewUser
		common.QuotaForInviter = oldQuotaForInviter
		common.QuotaForInvitee = oldQuotaForInvitee
		constant.GenerateDefaultToken = oldGenerateDefaultToken
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newInvitationContext(method string, target string, body []byte, remoteAddr string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	if body == nil {
		ctx.Request = httptest.NewRequest(method, target, nil)
	} else {
		ctx.Request = httptest.NewRequest(method, target, io.NopCloser(bytes.NewReader(body)))
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	ctx.Request.RemoteAddr = remoteAddr
	return ctx, recorder
}

func decodeInvitationResponse(t *testing.T, recorder *httptest.ResponseRecorder) invitationAPIResponse {
	t.Helper()

	var response invitationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func seedInvitationUser(t *testing.T, db *gorm.DB, user model.User) *model.User {
	t.Helper()
	require.NoError(t, db.Create(&user).Error)
	return &user
}

func TestGetSelf_HidesAffCodeWhenInvitationLinkSharingPaused(t *testing.T) {
	db := setupInvitationControllerTestDB(t)
	user := seedInvitationUser(t, db, model.User{
		Username:    "self-user",
		Password:    "hashed-password",
		DisplayName: "self-user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "ABCD",
	})

	ctx, recorder := newInvitationContext(http.MethodGet, "/api/user/self", nil, "198.51.100.10:1234")
	ctx.Set("id", user.Id)
	ctx.Set("role", common.RoleCommonUser)

	GetSelf(ctx)

	response := decodeInvitationResponse(t, recorder)
	require.True(t, response.Success)
	value, exists := response.Data["aff_code"]
	if exists {
		require.Empty(t, value)
	}
}

func TestRegister_IgnoresAffCodeWhenInvitationLinkSharingPaused(t *testing.T) {
	db := setupInvitationControllerTestDB(t)
	seedInvitationUser(t, db, model.User{
		Username:    "inviter",
		Password:    "hashed-password",
		DisplayName: "inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "AFF1",
	})

	payload, err := common.Marshal(map[string]any{
		"username": "invitee-user",
		"password": "password123",
		"aff_code": "AFF1",
	})
	require.NoError(t, err)

	ctx, recorder := newInvitationContext(http.MethodPost, "/api/user/register", payload, "198.51.100.11:1234")
	Register(ctx)

	response := decodeInvitationResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var createdUser model.User
	require.NoError(t, db.Where("username = ?", "invitee-user").First(&createdUser).Error)
	require.Equal(t, 0, createdUser.InviterId)
}

func TestGenerateOAuthCode_DoesNotStoreAffInSessionWhenInvitationLinkSharingPaused(t *testing.T) {
	setupInvitationControllerTestDB(t)
	ctx, recorder := newInvitationContext(http.MethodGet, "/api/oauth/state?aff=AFF1", nil, "198.51.100.12:1234")
	session := newFakeSession()
	ctx.Set(sessions.DefaultKey, session)

	GenerateOAuthCode(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Nil(t, session.Get("aff"))
	require.NotNil(t, session.Get("oauth_state"))
}

func TestFindOrCreateOAuthUser_IgnoresSessionAffWhenInvitationLinkSharingPaused(t *testing.T) {
	db := setupInvitationControllerTestDB(t)
	inviter := seedInvitationUser(t, db, model.User{
		Username:    "oauth-inviter",
		Password:    "hashed-password",
		DisplayName: "oauth-inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "AFF2",
	})
	common.QuotaForInvitee = 10
	common.QuotaForInviter = 20

	ctx, _ := newInvitationContext(http.MethodGet, "/api/oauth/fake/callback", nil, "198.51.100.13:1234")
	session := newFakeSession()
	session.Set("aff", "AFF2")

	user, err := findOrCreateOAuthUser(ctx, &fakeOAuthProvider{}, &oauth.OAuthUser{
		ProviderUserID: "provider-user-1",
		Username:       "oauth_invitee",
		DisplayName:    "oauth_invitee",
	}, session)
	require.NoError(t, err)

	createdUser, err := model.GetUserById(user.Id, true)
	require.NoError(t, err)
	require.Equal(t, 0, createdUser.Quota)

	reloadedInviter, err := model.GetUserById(inviter.Id, true)
	require.NoError(t, err)
	require.Equal(t, 0, reloadedInviter.AffCount)
	require.Equal(t, 0, reloadedInviter.AffQuota)
}
