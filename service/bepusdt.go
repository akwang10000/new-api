package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/shopspring/decimal"
)

type BEpusdtClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type BEpusdtUSDTNetwork struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Sort    int    `json:"sort"`
}

type BEpusdtCreateTransactionRequest struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	NotifyURL   string  `json:"notify_url"`
	RedirectURL string  `json:"redirect_url"`
	TradeType   string  `json:"trade_type"`
	Fiat        string  `json:"fiat"`
	Name        string  `json:"name,omitempty"`
	Timeout     int     `json:"timeout,omitempty"`
	Signature   string  `json:"signature"`
}

type BEpusdtCreateTransactionData struct {
	TradeID        string `json:"trade_id"`
	OrderID        string `json:"order_id"`
	Amount         string `json:"amount"`
	ActualAmount   string `json:"actual_amount"`
	Status         int    `json:"status"`
	Token          string `json:"token"`
	ExpirationTime int    `json:"expiration_time"`
	PaymentURL     string `json:"payment_url"`
}

type BEpusdtCreateTransactionResponse struct {
	StatusCode int                           `json:"status_code"`
	Message    string                        `json:"message"`
	Data       *BEpusdtCreateTransactionData `json:"data"`
}

type BEpusdtQueryOrderResponse struct {
	TradeID   string
	TradeHash string
	ReturnURL string
	Status    int
}

func ParseBEpusdtUSDTNetworks(raw string) ([]BEpusdtUSDTNetwork, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []BEpusdtUSDTNetwork{}, nil
	}
	var networks []BEpusdtUSDTNetwork
	if err := common.UnmarshalJsonStr(raw, &networks); err != nil {
		return nil, err
	}
	for i := range networks {
		networks[i].Code = strings.TrimSpace(networks[i].Code)
		networks[i].Name = strings.TrimSpace(networks[i].Name)
	}
	sort.SliceStable(networks, func(i, j int) bool {
		if networks[i].Sort == networks[j].Sort {
			return strings.ToLower(networks[i].Code) < strings.ToLower(networks[j].Code)
		}
		return networks[i].Sort < networks[j].Sort
	})
	return networks, nil
}

func GetEnabledBEpusdtNetworks() []BEpusdtUSDTNetwork {
	networks, err := ParseBEpusdtUSDTNetworks(setting.BEpusdtUSDTNetworks)
	if err != nil {
		return []BEpusdtUSDTNetwork{}
	}
	enabled := make([]BEpusdtUSDTNetwork, 0, len(networks))
	for _, network := range networks {
		if !network.Enabled || network.Code == "" {
			continue
		}
		enabled = append(enabled, network)
	}
	return enabled
}

func GetBEpusdtNetworkByCode(code string) (*BEpusdtUSDTNetwork, bool) {
	code = strings.TrimSpace(code)
	for _, network := range GetEnabledBEpusdtNetworks() {
		if strings.EqualFold(network.Code, code) {
			networkCopy := network
			return &networkCopy, true
		}
	}
	return nil, false
}

func IsBEpusdtEnabled() bool {
	return setting.BEpusdtEnabled &&
		strings.TrimSpace(setting.BEpusdtBaseURL) != "" &&
		strings.TrimSpace(setting.BEpusdtToken) != "" &&
		strings.TrimSpace(getBEpusdtWebhookSignSecret()) != "" &&
		len(GetEnabledBEpusdtNetworks()) > 0
}

func NewBEpusdtClient() *BEpusdtClient {
	if !IsBEpusdtEnabled() {
		return nil
	}
	httpClient := GetHttpClient()
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &BEpusdtClient{
		baseURL: strings.TrimRight(strings.TrimSpace(setting.BEpusdtBaseURL), "/"),
		token:   strings.TrimSpace(setting.BEpusdtToken),
		client:  httpClient,
	}
}

func (c *BEpusdtClient) CreateTransaction(reqBody *BEpusdtCreateTransactionRequest) (*BEpusdtCreateTransactionResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("bepusdt client is nil")
	}
	if reqBody == nil {
		return nil, fmt.Errorf("bepusdt create transaction request is nil")
	}
	var resp BEpusdtCreateTransactionResponse
	if err := c.doJSON(http.MethodPost, c.baseURL+"/api/v1/order/create-transaction", reqBody, &resp); err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 || resp.Data == nil {
		return nil, fmt.Errorf("bepusdt create transaction failed: %s", strings.TrimSpace(resp.Message))
	}
	return &resp, nil
}

func (c *BEpusdtClient) QueryOrder(tradeID string) (*BEpusdtQueryOrderResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("bepusdt client is nil")
	}
	tradeID = strings.TrimSpace(tradeID)
	if tradeID == "" {
		return nil, fmt.Errorf("bepusdt trade id is empty")
	}

	var rawResp map[string]any
	if err := c.doJSON(http.MethodGet, c.baseURL+"/pay/check-status/"+url.PathEscape(tradeID), nil, &rawResp); err != nil {
		return nil, err
	}
	remoteTradeID := strings.TrimSpace(common.Interface2String(rawResp["trade_id"]))
	if remoteTradeID == "" {
		return nil, fmt.Errorf("bepusdt query order returned empty trade_id")
	}
	status, err := parseBEpusdtStatusCode(rawResp["status"])
	if err != nil {
		return nil, err
	}
	return &BEpusdtQueryOrderResponse{
		TradeID:   remoteTradeID,
		TradeHash: strings.TrimSpace(common.Interface2String(rawResp["trade_hash"])),
		ReturnURL: strings.TrimSpace(common.Interface2String(rawResp["return_url"])),
		Status:    status,
	}, nil
}

func SignBEpusdtParams(params map[string]string) string {
	token := strings.TrimSpace(setting.BEpusdtToken)
	if token == "" {
		return ""
	}
	return signBEpusdtParamsWithToken(params, token)
}

func CanonicalBEpusdtDecimal(value float64, scale int32) string {
	return decimal.NewFromFloat(value).Round(scale).String()
}

func VerifyBEpusdtSignature(params map[string]string, signature string) bool {
	token := strings.TrimSpace(getBEpusdtWebhookSignSecret())
	signature = strings.TrimSpace(signature)
	if token == "" || signature == "" || len(params) == 0 {
		return false
	}
	expected := signBEpusdtParamsWithToken(params, token)
	if expected == "" {
		return false
	}
	return strings.EqualFold(expected, signature)
}

func getBEpusdtWebhookSignSecret() string {
	if secret := strings.TrimSpace(setting.BEpusdtWebhookSecret); secret != "" {
		return secret
	}
	return strings.TrimSpace(setting.BEpusdtToken)
}

func parseBEpusdtStatusCode(raw any) (int, error) {
	statusStr := strings.TrimSpace(common.Interface2String(raw))
	if statusStr == "" {
		return 0, fmt.Errorf("bepusdt query order returned empty status")
	}
	switch statusStr {
	case "1":
		return 1, nil
	case "2":
		return 2, nil
	case "3":
		return 3, nil
	default:
		return 0, fmt.Errorf("unsupported bepusdt status: %s", statusStr)
	}
}

func signBEpusdtParamsWithToken(params map[string]string, token string) string {
	filtered := make([]string, 0, len(params))
	for key, value := range params {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || strings.EqualFold(key, "signature") || value == "" {
			continue
		}
		filtered = append(filtered, key+"="+value)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		leftKey := strings.SplitN(filtered[i], "=", 2)[0]
		rightKey := strings.SplitN(filtered[j], "=", 2)[0]
		return leftKey < rightKey
	})
	payload := strings.Join(filtered, "&") + token
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func (c *BEpusdtClient) doJSON(method string, endpoint string, reqBody any, respBody any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		payload, err := common.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal bepusdt request failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(payload)
	}

	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create bepusdt request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send bepusdt request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read bepusdt response failed: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("bepusdt api status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}
	if respBody != nil && len(rawBody) > 0 {
		if err = common.Unmarshal(rawBody, respBody); err != nil {
			return fmt.Errorf("decode bepusdt response failed: %w", err)
		}
	}
	return nil
}
