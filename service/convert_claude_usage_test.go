package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestBuildClaudeUsageFromOpenAIUsageDefaultsAggregateCacheCreationTo5m(t *testing.T) {
	usage := buildClaudeUsageFromOpenAIUsage(&dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	})

	require.NotNil(t, usage)
	require.NotNil(t, usage.CacheCreation)
	require.EqualValues(t, 30, usage.CacheCreation.Ephemeral5mInputTokens)
	require.EqualValues(t, 20, usage.CacheCreation.Ephemeral1hInputTokens)
}

func TestStreamResponseOpenAI2ClaudeEmitsUsageOnlyFinalChunk(t *testing.T) {
	finishReason := "stop"
	info := &relaycommon.RelayInfo{
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			FinishReason: finishReason,
			Done:         false,
			Usage: &dto.Usage{
				PromptTokens:     100,
				CompletionTokens: 20,
				PromptTokensDetails: dto.InputTokenDetails{
					CachedTokens:         30,
					CachedCreationTokens: 50,
				},
				ClaudeCacheCreation5mTokens: 10,
				ClaudeCacheCreation1hTokens: 20,
			},
		},
	}

	responses := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{}, info)

	require.Len(t, responses, 2)
	require.Equal(t, "message_delta", responses[0].Type)
	require.NotNil(t, responses[0].Usage)
	require.EqualValues(t, 100, responses[0].Usage.InputTokens)
	require.EqualValues(t, 20, responses[0].Usage.OutputTokens)
	require.EqualValues(t, 30, responses[0].Usage.CacheReadInputTokens)
	require.EqualValues(t, 50, responses[0].Usage.CacheCreationInputTokens)
	require.NotNil(t, responses[0].Usage.CacheCreation)
	require.EqualValues(t, 30, responses[0].Usage.CacheCreation.Ephemeral5mInputTokens)
	require.EqualValues(t, 20, responses[0].Usage.CacheCreation.Ephemeral1hInputTokens)
	require.NotNil(t, responses[0].Delta)
	require.NotNil(t, responses[0].Delta.StopReason)
	require.Equal(t, "end_turn", *responses[0].Delta.StopReason)
	require.Equal(t, "message_stop", responses[1].Type)
	require.True(t, info.ClaudeConvertInfo.Done)
}

func TestStreamResponseOpenAI2ClaudeDefersDoneChunkUntilUsageArrives(t *testing.T) {
	finishReason := "stop"
	info := &relaycommon.RelayInfo{
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{},
	}

	responses := StreamResponseOpenAI2Claude(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			FinishReason: &finishReason,
		}},
	}, info)

	require.Empty(t, responses)
	require.Equal(t, "stop", info.FinishReason)
	require.False(t, info.ClaudeConvertInfo.Done)
}
