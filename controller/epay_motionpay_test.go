package controller

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
)

func TestMotionPayQRCodePreferred(t *testing.T) {
	args := &epay.PurchaseArgs{Type: "alipay"}
	if !motionPayQRCodePreferred(args, "pc") {
		t.Fatal("expected pc alipay checkout to prefer qrcode")
	}
	if motionPayQRCodePreferred(args, "mobile") {
		t.Fatal("expected mobile alipay checkout to keep direct links first")
	}
	if motionPayQRCodePreferred(&epay.PurchaseArgs{Type: "wxpay"}, "pc") {
		t.Fatal("expected non-alipay checkout not to prefer qrcode")
	}
}

func TestPickMotionPayCheckout(t *testing.T) {
	qr := newMotionPayCheckoutResponse("https://qr.alipay.com/test", "qrcode")
	url := newMotionPayCheckoutResponse("https://example.com/pay", "url")

	got := pickMotionPayCheckout(nil, qr, url)
	if got == nil || got.PayLink != qr.PayLink || got.QRContent != qr.PayLink {
		t.Fatalf("unexpected qrcode checkout: %#v", got)
	}

	got = pickMotionPayCheckout(nil, url)
	if got == nil || got.PayLink != url.PayLink || got.PayLinkType != "url" {
		t.Fatalf("unexpected url checkout: %#v", got)
	}
}

func TestValidateMotionPayCheckout(t *testing.T) {
	if err := validateMotionPayCheckout(&EpayCheckoutResponse{
		PayLink:     "https://qr.alipay.com/fkx123",
		PayLinkType: "qrcode",
		QRContent:   "https://qr.alipay.com/fkx123",
	}); err != nil {
		t.Fatalf("expected trusted alipay checkout, got error: %v", err)
	}

	if err := validateMotionPayCheckout(&EpayCheckoutResponse{
		PayLink:     "https://evil.example.com/pay",
		PayLinkType: "url",
	}); err == nil {
		t.Fatal("expected untrusted link to be rejected")
	}
}

func TestValidateMotionPayCheckoutAllowsQRCodePayload(t *testing.T) {
	checkout := &EpayCheckoutResponse{
		PayLink:     "https://checkout.motionpay.example/qrcode/abc123",
		PayLinkType: "qrcode",
		QRContent:   "https://checkout.motionpay.example/qrcode/abc123",
	}

	if err := validateMotionPayCheckout(checkout); err != nil {
		t.Fatalf("expected qrcode payload to be accepted, got error: %v", err)
	}
}

func TestValidateMotionPayCheckoutRejectsUntrustedPayLinkEvenForQRCode(t *testing.T) {
	checkout := &EpayCheckoutResponse{
		PayLink:     "https://motionpay.example/cashier.php?trade_no=123",
		PayLinkType: "qrcode",
		QRContent:   "https://motionpay.example/cashier.php?trade_no=123",
	}

	err := validateMotionPayCheckout(checkout)
	if err == nil {
		t.Fatal("expected cashier link to be rejected")
	}
	if !strings.Contains(err.Error(), "cashier page") {
		t.Fatalf("expected cashier page error, got: %v", err)
	}
}

func TestDecodeMotionPayMAPIResponseNonJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/html; charset=UTF-8"}},
		Body:       io.NopCloser(strings.NewReader("<html><body>blocked</body></html>")),
	}

	var payload motionPayMAPIResponse
	err := decodeMotionPayMAPIResponse(resp, &payload)
	if err == nil {
		t.Fatal("expected non-json response to return error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "non-JSON") || !strings.Contains(msg, "blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}
