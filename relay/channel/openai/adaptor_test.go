package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestAzureResponsesCompactRequestURLUsesResponsesCompactEndpoint(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/responses/compact",
		RelayMode:      relayconstant.RelayModeResponsesCompact,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAzure,
			ChannelBaseUrl:    "https://example.openai.azure.com",
			UpstreamModelName: "gpt-4.1",
		},
	}

	requestURL, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}
	want := "https://example.openai.azure.com/openai/v1/responses/compact?api-version=preview"
	if requestURL != want {
		t.Fatalf("requestURL = %q, want %q", requestURL, want)
	}
}

func TestAzureResponsesCompactRequestURLForCognitiveServicesUsesConfiguredVersion(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RequestURLPath: "/v1/responses/compact",
		RelayMode:      relayconstant.RelayModeResponsesCompact,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAzure,
			ChannelBaseUrl:    "https://example.cognitiveservices.azure.com",
			UpstreamModelName: "gpt-4.1",
			ApiVersion:        "2025-04-01-preview",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AzureResponsesVersion: "2026-01-01-preview",
			},
		},
	}

	requestURL, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}
	want := "https://example.cognitiveservices.azure.com/openai/responses/compact?api-version=2026-01-01-preview"
	if requestURL != want {
		t.Fatalf("requestURL = %q, want %q", requestURL, want)
	}
}
