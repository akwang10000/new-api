package vertex

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestBuildGoogleModelURLDefaultGlobalWithoutProject(t *testing.T) {
	got := BuildGoogleModelURL("", DefaultAPIVersion, "", "global", "gemini-2.5-pro", "generateContent")
	want := "https://aiplatform.googleapis.com/v1/publishers/google/models/gemini-2.5-pro:generateContent"
	if got != want {
		t.Fatalf("BuildGoogleModelURL() = %s, want %s", got, want)
	}
}

func TestBuildGoogleModelURLDefaultRegionalWithProject(t *testing.T) {
	got := BuildGoogleModelURL("", DefaultAPIVersion, "project-1", "us-central1", "gemini-2.5-pro", "streamGenerateContent?alt=sse")
	want := "https://us-central1-aiplatform.googleapis.com/v1/projects/project-1/locations/us-central1/publishers/google/models/gemini-2.5-pro:streamGenerateContent?alt=sse"
	if got != want {
		t.Fatalf("BuildGoogleModelURL() = %s, want %s", got, want)
	}
}

func TestBuildAnthropicModelURLCustomBaseWithProject(t *testing.T) {
	got := BuildAnthropicModelURL("https://vertex-gateway.example.com/proxy", DefaultAPIVersion, "project-1", "europe-west1", "claude-sonnet-4@20250514", "rawPredict")
	want := "https://vertex-gateway.example.com/proxy/v1/projects/project-1/locations/europe-west1/publishers/anthropic/models/claude-sonnet-4@20250514:rawPredict"
	if got != want {
		t.Fatalf("BuildAnthropicModelURL() = %s, want %s", got, want)
	}
}

func TestBuildAPIBaseURLDoesNotDuplicateVersion(t *testing.T) {
	got := BuildAPIBaseURL("https://vertex-gateway.example.com/proxy/v1/", DefaultAPIVersion, "project-1", "global")
	want := "https://vertex-gateway.example.com/proxy/v1/projects/project-1/locations/global"
	if got != want {
		t.Fatalf("BuildAPIBaseURL() = %s, want %s", got, want)
	}
}

func TestBuildOpenSourceChatCompletionsURLCustomBase(t *testing.T) {
	got := BuildOpenSourceChatCompletionsURL("https://vertex-gateway.example.com", "project-1", "us-central1")
	want := "https://vertex-gateway.example.com/v1beta1/projects/project-1/locations/us-central1/endpoints/openapi/chat/completions"
	if got != want {
		t.Fatalf("BuildOpenSourceChatCompletionsURL() = %s, want %s", got, want)
	}
}

func TestAdaptorGetRequestURLUsesCustomBaseURLForServiceAccountGemini(t *testing.T) {
	apiKey, err := common.Marshal(Credentials{ProjectID: "project-1"})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	info := &relaycommon.RelayInfo{
		IsStream:        false,
		OriginModelName: "gemini-2.5-pro",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            string(apiKey),
			ApiVersion:        "us-central1",
			ChannelBaseUrl:    "https://vertex-gateway.example.com/proxy",
			UpstreamModelName: "gemini-2.5-pro",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				VertexKeyType: dto.VertexKeyTypeJSON,
			},
		},
	}
	adaptor := &Adaptor{RequestMode: RequestModeGemini}

	got, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}
	want := "https://vertex-gateway.example.com/proxy/v1/projects/project-1/locations/us-central1/publishers/google/models/gemini-2.5-pro:generateContent"
	if got != want {
		t.Fatalf("GetRequestURL() = %s, want %s", got, want)
	}
}
