package claude

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestAdaptorGetRequestURLUsesCountTokensPath(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeClaudeCountTokens,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.anthropic.com",
		},
	}

	requestURL, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}

	if requestURL != "https://api.anthropic.com/v1/messages/count_tokens" {
		t.Fatalf("requestURL = %q, want count_tokens path", requestURL)
	}
}
