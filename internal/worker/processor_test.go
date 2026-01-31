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

func TestProcessor_GetModelConfig_MultipleModels(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "gpt-3.5-turbo", APIProvider: "openai", APIKey: "key1"},
			{Name: "gpt-4", APIProvider: "openai", APIKey: "key2"},
			{Name: "gpt-4-turbo", APIProvider: "openai", APIKey: "key3"},
			{Name: "claude-3-opus", APIProvider: "anthropic", APIKey: "key4"},
			{Name: "claude-3-sonnet", APIProvider: "anthropic", APIKey: "key5"},
		},
	}

	processor := &Processor{cfg: cfg}

	// Test first model
	provider, apiKey := processor.getModelConfig("gpt-3.5-turbo")
	assert.Equal(t, "openai", provider)
	assert.Equal(t, "key1", apiKey)

	// Test middle model
	provider, apiKey = processor.getModelConfig("gpt-4-turbo")
	assert.Equal(t, "openai", provider)
	assert.Equal(t, "key3", apiKey)

	// Test last model
	provider, apiKey = processor.getModelConfig("claude-3-sonnet")
	assert.Equal(t, "anthropic", provider)
	assert.Equal(t, "key5", apiKey)
}

func TestProcessor_GetModelConfig_CaseSensitive(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "GPT-4", APIProvider: "openai", APIKey: "key1"},
		},
	}

	processor := &Processor{cfg: cfg}

	// Exact match should work
	provider, apiKey := processor.getModelConfig("GPT-4")
	assert.Equal(t, "openai", provider)
	assert.Equal(t, "key1", apiKey)

	// Different case should not match
	provider, apiKey = processor.getModelConfig("gpt-4")
	assert.Empty(t, provider)
	assert.Empty(t, apiKey)
}

func TestProcessor_GetModelConfig_EmptyAPIKey(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "free-model", APIProvider: "local", APIKey: ""},
		},
	}

	processor := &Processor{cfg: cfg}

	provider, apiKey := processor.getModelConfig("free-model")
	assert.Equal(t, "local", provider)
	assert.Empty(t, apiKey) // Empty API key is valid for some local models
}

func TestProcessor_GetModelConfig_SpecialCharacters(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "model-with-special_chars.v1", APIProvider: "custom", APIKey: "key"},
		},
	}

	processor := &Processor{cfg: cfg}

	provider, apiKey := processor.getModelConfig("model-with-special_chars.v1")
	assert.Equal(t, "custom", provider)
	assert.Equal(t, "key", apiKey)
}

func TestNewProcessor_WithAllNilDeps(t *testing.T) {
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "test-model", APIProvider: "test", APIKey: "test-key"},
		},
	}

	processor := NewProcessor(nil, nil, nil, nil, cfg)

	assert.NotNil(t, processor)
	assert.Nil(t, processor.jobRepo)
	assert.Nil(t, processor.analysisRepo)
	assert.Nil(t, processor.ossClient)
	assert.Nil(t, processor.publisher)
	assert.Equal(t, cfg, processor.cfg)

	// Should still be able to get model config
	provider, _ := processor.getModelConfig("test-model")
	assert.Equal(t, "test", provider)
}

func TestProcessor_GetModelConfig_FirstMatchWins(t *testing.T) {
	// If there are duplicate model names (shouldn't happen but test behavior)
	cfg := &config.Config{
		Models: []config.ModelConfig{
			{Name: "gpt-4", APIProvider: "first", APIKey: "key1"},
			{Name: "gpt-4", APIProvider: "second", APIKey: "key2"},
		},
	}

	processor := &Processor{cfg: cfg}

	// Should return first match
	provider, apiKey := processor.getModelConfig("gpt-4")
	assert.Equal(t, "first", provider)
	assert.Equal(t, "key1", apiKey)
}
