package authentication

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Rahmannugar/authlier/sessiontoken"
)

const sessionCacheMaximumJitter = 10

// JitteredSessionCache spreads cache expiry while never extending an entry
// beyond the lifetime selected by Authlier.
type JitteredSessionCache struct {
	cache sessiontoken.Cache
}

func NewJitteredSessionCache(cache sessiontoken.Cache) (*JitteredSessionCache, error) {
	if cache == nil {
		return nil, fmt.Errorf("session cache is required")
	}
	return &JitteredSessionCache{cache: cache}, nil
}

func (cache *JitteredSessionCache) Get(
	ctx context.Context,
	tokenHash sessiontoken.TokenHash,
) (sessiontoken.Record, error) {
	return cache.cache.Get(ctx, tokenHash)
}

func (cache *JitteredSessionCache) Set(
	ctx context.Context,
	record sessiontoken.Record,
	ttl time.Duration,
) error {
	return cache.cache.Set(ctx, record, jitteredTTL(ttl))
}

func (cache *JitteredSessionCache) Delete(
	ctx context.Context,
	tokenHash sessiontoken.TokenHash,
) error {
	return cache.cache.Delete(ctx, tokenHash)
}

func jitteredTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}

	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ttl
	}
	percent := binary.LittleEndian.Uint64(value[:]) % (sessionCacheMaximumJitter + 1)
	reduction := ttl * time.Duration(percent) / 100
	return ttl - reduction
}

var _ sessiontoken.Cache = (*JitteredSessionCache)(nil)
