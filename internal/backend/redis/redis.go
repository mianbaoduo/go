package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kellegous/go/internal"
	"github.com/kellegous/go/internal/backend"
	"github.com/redis/go-redis/v9"
)

var _ backend.Backend = (*Backend)(nil)

// Backend provides access to Redis store.
type Backend struct {
	client *redis.Client
	prefix string
}

// New creates a new Redis backend.
func New(addr, password string, db int, prefix string) (*Backend, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Backend{
		client: client,
		prefix: prefix,
	}, nil
}

// Close closes the Redis connection.
func (b *Backend) Close() error {
	return b.client.Close()
}

// Get retrieves a route by name.
func (b *Backend) Get(ctx context.Context, name string) (*internal.Route, error) {
	key := b.key(name)
	val, err := b.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, internal.ErrRouteNotFound
		}
		return nil, err
	}

	var route internal.Route
	if err := json.Unmarshal(val, &route); err != nil {
		return nil, err
	}

	return &route, nil
}

// Put stores a route with the given name.
func (b *Backend) Put(ctx context.Context, name string, route *internal.Route) error {
	key := b.key(name)
	val, err := json.Marshal(route)
	if err != nil {
		return err
	}

	return b.client.Set(ctx, key, val, 0).Err()
}

// Del removes a route by name.
func (b *Backend) Del(ctx context.Context, name string) error {
	key := b.key(name)
	return b.client.Del(ctx, key).Err()
}

// List returns an iterator for all routes starting with the given prefix.
func (b *Backend) List(ctx context.Context, start string) (internal.RouteIterator, error) {
	pattern := b.key("*")
	keys, err := b.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	// Filter keys that start with the given prefix
	var filteredKeys []string
	for _, key := range keys {
		if start == "" || key >= b.key(start) {
			filteredKeys = append(filteredKeys, key)
		}
	}

	return &RouteIterator{
		ctx:    ctx,
		client: b.client,
		keys:   filteredKeys,
		index:  -1,
	}, nil
}

// GetAll returns all routes in the database.
func (b *Backend) GetAll(ctx context.Context) (map[string]internal.Route, error) {
	pattern := b.key("*")
	keys, err := b.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	routes := make(map[string]internal.Route)
	for _, key := range keys {
		val, err := b.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var route internal.Route
		if err := json.Unmarshal(val, &route); err != nil {
			continue
		}

		name := b.unkey(key)
		routes[name] = route
	}

	return routes, nil
}

// NextID generates the next numeric ID for auto-generated routes.
func (b *Backend) NextID(ctx context.Context) (uint64, error) {
	key := b.key("next_id")
	return b.client.Incr(ctx, key).Uint64()
}

// key returns the Redis key for the given route name.
func (b *Backend) key(name string) string {
	return fmt.Sprintf("%s:%s", b.prefix, name)
}

// unkey extracts the route name from a Redis key.
func (b *Backend) unkey(key string) string {
	return key[len(b.prefix)+1:]
}

// RouteIterator implements the internal.RouteIterator interface.
type RouteIterator struct {
	ctx    context.Context
	client *redis.Client
	keys   []string
	index  int
	route  *internal.Route
}

// Valid returns true if the iterator is valid.
func (i *RouteIterator) Valid() bool {
	return i.index >= 0 && i.index < len(i.keys)
}

// Next advances the iterator to the next route.
func (i *RouteIterator) Next() bool {
	i.index++
	if !i.Valid() {
		return false
	}

	val, err := i.client.Get(i.ctx, i.keys[i.index]).Bytes()
	if err != nil {
		return false
	}

	var route internal.Route
	if err := json.Unmarshal(val, &route); err != nil {
		return false
	}

	i.route = &route
	return true
}

// Seek moves the iterator to the first route with a name >= the given name.
func (i *RouteIterator) Seek(name string) bool {
	for i.index = 0; i.index < len(i.keys); i.index++ {
		if i.keys[i.index] >= name {
			return i.Next()
		}
	}
	return false
}

// Error returns any error that occurred during iteration.
func (i *RouteIterator) Error() error {
	return nil
}

// Name returns the name of the current route.
func (i *RouteIterator) Name() string {
	if !i.Valid() {
		return ""
	}
	return i.keys[i.index][len("go:routes:"):]
}

// Route returns the current route.
func (i *RouteIterator) Route() *internal.Route {
	return i.route
}

// Release releases any resources associated with the iterator.
func (i *RouteIterator) Release() {
	// No resources to release
}
