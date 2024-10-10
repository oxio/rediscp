package rediscp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient Function to create a new Redis client connection
func NewRedisClient(url string) *redis.Client {
	opts, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	return redis.NewClient(opts)
}

type RedisScanner struct {
}

type RedisCopier struct {
	IgnoreTTL       bool
	SkipExisting    bool
	ReplaceExisting bool
	Verbose         bool
}

type ErrBusykey struct {
	Key string
}

var _ error = ErrBusykey{}

func (e ErrBusykey) Error() string {
	return fmt.Sprintf("key \"%s\" already exists in target Redis", e.Key)
}

// ScanKeys Function to scan for keys matching a pattern in Redis
func (c *RedisScanner) ScanKeys(ctx context.Context, client *redis.Client, pattern string) ([]string, error) {
	var cursor uint64
	var keys []string

	for {
		var err error
		var scannedKeys []string
		scannedKeys, cursor, err = client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, scannedKeys...)

		if cursor == 0 { // Finished scanning all keys
			break
		}
	}

	return keys, nil
}

// CopyKeys Function to copy keys from source Redis to target Redis
func (c *RedisCopier) CopyKeys(ctx context.Context, sourceClient, targetClient *redis.Client, keys []string) (copied int, skipped int, err error) {
	for _, key := range keys {
		// Get the value of the key from the source Redis
		val, err := sourceClient.Dump(ctx, key).Result()
		if err != nil {
			return copied, skipped, fmt.Errorf("failed to dump key %s from source Redis: %w", key, err)
		}

		// Get the TTL (time to live) of the key from the source Redis
		var ttl time.Duration
		if c.IgnoreTTL {
			ttl = time.Duration(0)
		} else {
			ttl, err = sourceClient.TTL(ctx, key).Result()
			if err != nil {
				return copied, skipped, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
			}
		}

		// Restore the key with the TTL to the target Redis
		if c.ReplaceExisting {
			err = targetClient.RestoreReplace(ctx, key, ttl, val).Err()
		} else {
			err = targetClient.Restore(ctx, key, ttl, val).Err()
		}

		if err != nil {
			if strings.HasPrefix(err.Error(), "BUSYKEY") {
				if c.SkipExisting {
					if c.Verbose {
						fmt.Printf("Key already exists (skipping): %s\n", key)
					}
					skipped++
					continue
				}
				return copied, skipped, &ErrBusykey{Key: key}
			}
			return copied, skipped, fmt.Errorf("failed to restore key %s to target Redis: %w", key, err)
		}

		if c.Verbose {
			fmt.Printf("Copied key: %s\n", key)
		}
		copied++
	}

	return copied, skipped, nil
}
