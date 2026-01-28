package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/qs3c/anal_go_server/config"
)

func TestNewProcessor(t *testing.T) {
	cfg := &config.Config{}

	// Test that NewProcessor doesn't panic with nil dependencies
	// In production, dependencies would be properly initialized
	processor := NewProcessor(nil, nil, nil, nil, cfg)

	assert.NotNil(t, processor)
	assert.Equal(t, cfg, processor.cfg)
}

func TestProcessor_GetModelConfig(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{
				Name:        "gpt-4",
				APIProvider: "openai",
				APIKey:      "sk-test-openai",
			},
			{
				Name:        "claude-3",
				APIProvider: "anthropic",
				APIKey:      "sk-test-anthropic",
			},
		},
	}

	processor := &Processor{cfg: cfg}

	tests := []struct {
		name         string
		modelName    string
		wantProvider string
		wantAPIKey   string
	}{
		{
			name:         "existing model gpt-4",
			modelName:    "gpt-4",
			wantProvider: "openai",
			wantAPIKey:   "sk-test-openai",
		},
		{
			name:         "existing model claude-3",
			modelName:    "claude-3",
			wantProvider: "anthropic",
			wantAPIKey:   "sk-test-anthropic",
		},
		{
			name:         "non-existing model",
			modelName:    "unknown-model",
			wantProvider: "",
			wantAPIKey:   "",
		},
		{
			name:         "empty model name",
			modelName:    "",
			wantProvider: "",
			wantAPIKey:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, apiKey := processor.getModelConfig(tt.modelName)
			assert.Equal(t, tt.wantProvider, provider)
			assert.Equal(t, tt.wantAPIKey, apiKey)
		})
	}
}

func TestProcessor_GetModelConfig_EmptyModels(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{},
	}

	processor := &Processor{cfg: cfg}

	provider, apiKey := processor.getModelConfig("any-model")
	assert.Empty(t, provider)
	assert.Empty(t, apiKey)
}

func TestProcessor_GetModelConfig_NilConfig(t *testing.T) {
	// Test with nil Models slice
	cfg := &config.Config{}
	processor := &Processor{cfg: cfg}

	provider, apiKey := processor.getModelConfig("any-model")
	assert.Empty(t, provider)
	assert.Empty(t, apiKey)
}
