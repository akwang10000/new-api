package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

type BTCPayClient struct {
	baseURL string
	storeID string
	token   string
	client  *http.Client
}

type BTCPayInvoiceCheckout struct {
	RedirectURL           string `json:"redirectURL,omitempty"`
	RedirectAutomatically bool   `json:"redirectAutomatically,omitempty"`
}

type BTCPayCreateInvoiceRequest struct {
	Amount   float64                `json:"amount"`
	Currency string                 `json:"currency"`
	Metadata map[string]any         `json:"metadata,omitempty"`
	Checkout *BTCPayInvoiceCheckout `json:"checkout,omitempty"`
}

type BTCPayInvoiceResponse struct {
	ID           string         `json:"id"`
	Amount       any            `json:"amount"`
	Currency     string         `json:"currency"`
	CheckoutLink string         `json:"checkoutLink"`
	Status       string         `json:"status"`
	Metadata     map[string]any `json:"metadata"`
}

type BTCPayWebhookEvent struct {
	Type      string `json:"type"`
	InvoiceID string `json:"invoiceId"`
	StoreID   string `json:"storeId"`
}

func IsBTCPayEnabled() bool {
	return setting.BTCPayEnabled &&
		strings.TrimSpace(setting.BTCPayServerURL) != "" &&
		strings.TrimSpace(setting.BTCPayStoreID) != "" &&
		strings.TrimSpace(setting.BTCPayApiToken) != "" &&
		strings.TrimSpace(setting.BTCPayWebhookSecret) != ""
}

func NewBTCPayClient() *BTCPayClient {
	if !IsBTCPayEnabled() {
		return nil
	}
	httpClient := GetHttpClient()
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &BTCPayClient{
		baseURL: strings.TrimRight(strings.TrimSpace(setting.BTCPayServerURL), "/"),
		storeID: strings.TrimSpace(setting.BTCPayStoreID),
		token:   strings.TrimSpace(setting.BTCPayApiToken),
		client:  httpClient,
	}
}

func (c *BTCPayClient) CreateInvoice(reqBody *BTCPayCreateInvoiceRequest) (*BTCPayInvoiceResponse, error) {
	if reqBody == nil {
		return nil, fmt.Errorf("btcpay create invoice request is nil")
	}
	var resp BTCPayInvoiceResponse
	if err := c.doJSON(http.MethodPost, c.invoiceAPIPath(""), reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *BTCPayClient) GetInvoice(invoiceID string) (*BTCPayInvoiceResponse, error) {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, fmt.Errorf("btcpay invoice id is empty")
	}
	var resp BTCPayInvoiceResponse
	if err := c.doJSON(http.MethodGet, c.invoiceAPIPath(invoiceID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func VerifyBTCPaySignature(payload []byte, signature string) bool {
	secret := strings.TrimSpace(setting.BTCPayWebhookSecret)
	if secret == "" || len(payload) == 0 {
		return false
	}
	signature = strings.TrimSpace(signature)
	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(signature)))
}

func (c *BTCPayClient) invoiceAPIPath(invoiceID string) string {
	base := fmt.Sprintf("%s/api/v1/stores/%s/invoices", c.baseURL, url.PathEscape(c.storeID))
	if strings.TrimSpace(invoiceID) == "" {
		return base
	}
	return base + "/" + url.PathEscape(strings.TrimSpace(invoiceID))
}

func (c *BTCPayClient) doJSON(method string, endpoint string, reqBody any, respBody any) error {
	if c == nil {
		return fmt.Errorf("btcpay client is nil")
	}
	var bodyReader io.Reader
	if reqBody != nil {
		payload, err := common.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal btcpay request failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(payload)
	}
	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create btcpay request failed: %w", err)
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send btcpay request failed: %w", err)
	}
	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read btcpay response failed: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("btcpay api status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}
	if respBody != nil && len(rawBody) > 0 {
		if err = common.Unmarshal(rawBody, respBody); err != nil {
			return fmt.Errorf("decode btcpay response failed: %w", err)
		}
	}
	return nil
}
