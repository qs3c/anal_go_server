package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	stateKeyPrefix = "oauth:state:"
	stateTTL       = 10 * time.Minute
)

// StateStore handles OAuth state parameter storage and validation
type StateStore struct {
	rdb *redis.Client
}

// NewStateStore creates a new StateStore
func NewStateStore(rdb *redis.Client) *StateStore {
	return &StateStore{rdb: rdb}
}

// StateData holds the data associated with an OAuth state
type StateData struct {
	RedirectURI string `json:"redirect_uri"`
}

// GenerateState creates a new cryptographically secure state token
// and stores the associated redirect URI in Redis
func (s *StateStore) GenerateState(ctx context.Context, redirectURI string) (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	state := hex.EncodeToString(bytes)

	// Store state with redirect URI in Redis
	key := stateKeyPrefix + state
	if err := s.rdb.Set(ctx, key, redirectURI, stateTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	return state, nil
}

// ValidateState checks if the state is valid and returns the associated redirect URI
// The state is consumed (deleted) after validation to prevent replay attacks
func (s *StateStore) ValidateState(ctx context.Context, state string) (string, error) {
	if state == "" {
		return "", fmt.Errorf("empty state parameter")
	}

	key := stateKeyPrefix + state

	// Get and delete atomically using a transaction
	var redirectURI string
	err := s.rdb.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			return fmt.Errorf("invalid or expired state")
		}
		if err != nil {
			return fmt.Errorf("failed to get state: %w", err)
		}
		redirectURI = val

		// Delete the state to prevent reuse
		_, err = tx.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			return nil
		})
		return err
	}, key)

	if err != nil {
		return "", err
	}

	return redirectURI, nil
}
