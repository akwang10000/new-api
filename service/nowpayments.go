package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

const nowPaymentsAPIBaseURL = "https://api.nowpayments.io/v1"

type NOWPaymentsClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type NOWPaymentsUSDTNetwork struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Sort    int    `json:"sort"`
}

type NOWPaymentsMinAmountResponse struct {
	MinAmount NOWPaymentsFloat64 `json:"min_amount"`
}

type NOWPaymentsCreateInvoiceRequest struct {
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayCurrency      string  `json:"pay_currency"`
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description,omitempty"`
	IPNCallbackURL   string  `json:"ipn_callback_url,omitempty"`
	SuccessURL       string  `json:"success_url,omitempty"`
	CancelURL        string  `json:"cancel_url,omitempty"`
}

type NOWPaymentsInvoiceResponse struct {
	ID            any                `json:"id"`
	OrderID       string             `json:"order_id"`
	InvoiceURL    string             `json:"invoice_url"`
	PayCurrency   string             `json:"pay_currency"`
	PriceCurrency string             `json:"price_currency"`
	PriceAmount   NOWPaymentsFloat64 `json:"price_amount"`
}

type NOWPaymentsPaymentResponse struct {
	PaymentID      any                `json:"payment_id"`
	OrderID        string             `json:"order_id"`
	PaymentStatus  string             `json:"payment_status"`
	PayCurrency    string             `json:"pay_currency"`
	PriceCurrency  string             `json:"price_currency"`
	PriceAmount    NOWPaymentsFloat64 `json:"price_amount"`
	ActuallyPaid   NOWPaymentsFloat64 `json:"actually_paid"`
	ActuallyPaidAt NOWPaymentsFloat64 `json:"actually_paid_at_fiat"`
}

type NOWPaymentsFloat64 float64

func (n *NOWPaymentsFloat64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*n = 0
		return nil
	}

	switch common.GetJsonType(json.RawMessage(data)) {
	case "number":
		var value float64
		if err := common.Unmarshal(data, &value); err != nil {
			return err
		}
		*n = NOWPaymentsFloat64(value)
		return nil
	case "string":
		var raw string
		if err := common.Unmarshal(data, &raw); err != nil {
			return err
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			*n = 0
			return nil
		}
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		*n = NOWPaymentsFloat64(value)
		return nil
	default:
		return fmt.Errorf("unsupported nowpayments numeric value: %s", string(data))
	}
}

func ParseNOWPaymentsUSDTNetworks(raw string) ([]NOWPaymentsUSDTNetwork, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []NOWPaymentsUSDTNetwork{}, nil
	}
	var networks []NOWPaymentsUSDTNetwork
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

func GetEnabledNOWPaymentsNetworks() []NOWPaymentsUSDTNetwork {
	networks, err := ParseNOWPaymentsUSDTNetworks(setting.NOWPaymentsUSDTNetworks)
	if err != nil {
		return []NOWPaymentsUSDTNetwork{}
	}
	enabled := make([]NOWPaymentsUSDTNetwork, 0, len(networks))
	for _, network := range networks {
		if !network.Enabled || network.Code == "" {
			continue
		}
		enabled = append(enabled, network)
	}
	return enabled
}

func GetNOWPaymentsNetworkByCode(code string) (*NOWPaymentsUSDTNetwork, bool) {
	code = strings.TrimSpace(code)
	for _, network := range GetEnabledNOWPaymentsNetworks() {
		if strings.EqualFold(network.Code, code) {
			networkCopy := network
			return &networkCopy, true
		}
	}
	return nil, false
}

func ParseNOWPaymentsCryptoAmountOptions(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}, nil
	}
	var options []int
	if err := common.UnmarshalJsonStr(raw, &options); err != nil {
		return nil, err
	}
	return options, nil
}

func IsNOWPaymentsEnabled() bool {
	return setting.NOWPaymentsEnabled &&
		strings.TrimSpace(setting.NOWPaymentsApiKey) != "" &&
		strings.TrimSpace(setting.NOWPaymentsIPNSecret) != "" &&
		(setting.NOWPaymentsFiatModeEnabled || setting.NOWPaymentsCryptoModeEnabled) &&
		len(GetEnabledNOWPaymentsNetworks()) > 0
}

func IsNOWPaymentsModeEnabled(pricingMode string) bool {
	switch strings.ToLower(strings.TrimSpace(pricingMode)) {
	case "fiat":
		return setting.NOWPaymentsFiatModeEnabled
	case "crypto":
		return setting.NOWPaymentsCryptoModeEnabled
	default:
		return false
	}
}

func NewNOWPaymentsClient() *NOWPaymentsClient {
	if !IsNOWPaymentsEnabled() {
		return nil
	}
	httpClient := GetHttpClient()
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &NOWPaymentsClient{
		baseURL: nowPaymentsAPIBaseURL,
		apiKey:  strings.TrimSpace(setting.NOWPaymentsApiKey),
		client:  httpClient,
	}
}

func (c *NOWPaymentsClient) GetMinAmount(fromCurrency string, toCurrency string) (*NOWPaymentsMinAmountResponse, error) {
	params := url.Values{}
	params.Set("currency_from", strings.TrimSpace(fromCurrency))
	params.Set("currency_to", strings.TrimSpace(toCurrency))

	var resp NOWPaymentsMinAmountResponse
	if err := c.doJSON(http.MethodGet, c.baseURL+"/min-amount?"+params.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *NOWPaymentsClient) CreateInvoice(reqBody *NOWPaymentsCreateInvoiceRequest) (*NOWPaymentsInvoiceResponse, error) {
	if reqBody == nil {
		return nil, fmt.Errorf("nowpayments create invoice request is nil")
	}
	var resp NOWPaymentsInvoiceResponse
	if err := c.doJSON(http.MethodPost, c.baseURL+"/invoice", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *NOWPaymentsClient) GetPayment(paymentID string) (*NOWPaymentsPaymentResponse, error) {
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, fmt.Errorf("nowpayments payment id is empty")
	}
	var resp NOWPaymentsPaymentResponse
	if err := c.doJSON(http.MethodGet, c.baseURL+"/payment/"+url.PathEscape(paymentID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func VerifyNOWPaymentsSignature(payload []byte, signature string) bool {
	secret := strings.TrimSpace(setting.NOWPaymentsIPNSecret)
	signature = strings.TrimSpace(signature)
	if secret == "" || len(payload) == 0 || signature == "" {
		return false
	}

	canonicalPayload, err := canonicalizeNOWPaymentsPayload(payload)
	if err != nil {
		return false
	}

	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(canonicalPayload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(signature)))
}

func canonicalizeNOWPaymentsPayload(payload []byte) ([]byte, error) {
	return canonicalizeNOWPaymentsJSON(json.RawMessage(bytes.TrimSpace(payload)))
}

func canonicalizeNOWPaymentsJSON(raw json.RawMessage) ([]byte, error) {
	raw = json.RawMessage(bytes.TrimSpace(raw))
	switch common.GetJsonType(raw) {
	case "object":
		var obj map[string]json.RawMessage
		if err := common.Unmarshal(raw, &obj); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(obj))
		for key := range obj {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyJSON, err := common.Marshal(key)
			if err != nil {
				return nil, err
			}
			valueJSON, err := canonicalizeNOWPaymentsJSON(obj[key])
			if err != nil {
				return nil, err
			}
			buf.Write(keyJSON)
			buf.WriteByte(':')
			buf.Write(valueJSON)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil
	case "array":
		var arr []json.RawMessage
		if err := common.Unmarshal(raw, &arr); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i := range arr {
			if i > 0 {
				buf.WriteByte(',')
			}
			valueJSON, err := canonicalizeNOWPaymentsJSON(arr[i])
			if err != nil {
				return nil, err
			}
			buf.Write(valueJSON)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	default:
		return bytes.TrimSpace(raw), nil
	}
}

func FormatNOWPaymentsPaymentID(paymentID any) string {
	switch typed := paymentID.(type) {
	case string:
		return strings.TrimSpace(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return strings.TrimSpace(fmt.Sprint(paymentID))
	}
}

func (c *NOWPaymentsClient) doJSON(method string, endpoint string, reqBody any, respBody any) error {
	if c == nil {
		return fmt.Errorf("nowpayments client is nil")
	}
	var bodyReader io.Reader
	if reqBody != nil {
		payload, err := common.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal nowpayments request failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(payload)
	}
	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create nowpayments request failed: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send nowpayments request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read nowpayments response failed: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("nowpayments api status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}
	if respBody != nil && len(rawBody) > 0 {
		if err = common.Unmarshal(rawBody, respBody); err != nil {
			return fmt.Errorf("decode nowpayments response failed: %w", err)
		}
	}
	return nil
}
