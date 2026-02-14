package service

import (
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

// mockCloseableConfig implements ConfigService with a configurable Close error.
type mockCloseableConfig struct {
	closeErr error
}

func (m *mockCloseableConfig) Close() error                      { return m.closeErr }
func (m *mockCloseableConfig) ListConnections() ([]types.Connection, error) { return nil, nil }
func (m *mockCloseableConfig) AddConnection(_, _ string, _ int, _ string, _ int, _ bool) (types.Connection, error) {
	return types.Connection{}, nil
}
func (m *mockCloseableConfig) UpdateConnection(_ int64, _, _ string, _ int, _ string, _ int, _ bool) (types.Connection, error) {
	return types.Connection{}, nil
}
func (m *mockCloseableConfig) DeleteConnection(_ int64) error                   { return nil }
func (m *mockCloseableConfig) AddFavorite(_ int64, _, _ string) (types.Favorite, error) {
	return types.Favorite{}, nil
}
func (m *mockCloseableConfig) RemoveFavorite(_ int64, _ string) error           { return nil }
func (m *mockCloseableConfig) ListFavorites(_ int64) []types.Favorite           { return nil }
func (m *mockCloseableConfig) IsFavorite(_ int64, _ string) bool                { return false }
func (m *mockCloseableConfig) AddRecentKey(_ int64, _ string, _ types.KeyType)  {}
func (m *mockCloseableConfig) ListRecentKeys(_ int64) []types.RecentKey         { return nil }
func (m *mockCloseableConfig) ClearRecentKeys(_ int64)                          {}
func (m *mockCloseableConfig) AddValueHistory(_ string, _ types.RedisValue, _ string) {}
func (m *mockCloseableConfig) GetValueHistory(_ string) []types.ValueHistoryEntry { return nil }
func (m *mockCloseableConfig) ClearValueHistory()                               {}
func (m *mockCloseableConfig) ListTemplates() []types.KeyTemplate               { return nil }
func (m *mockCloseableConfig) AddTemplate(_ types.KeyTemplate) error            { return nil }
func (m *mockCloseableConfig) DeleteTemplate(_ string) error                    { return nil }
func (m *mockCloseableConfig) ListGroups() []types.ConnectionGroup              { return nil }
func (m *mockCloseableConfig) AddGroup(_, _ string) error                       { return nil }
func (m *mockCloseableConfig) AddConnectionToGroup(_ string, _ int64) error     { return nil }
func (m *mockCloseableConfig) RemoveConnectionFromGroup(_ string, _ int64) error { return nil }
func (m *mockCloseableConfig) GetKeyBindings() types.KeyBindings                { return types.KeyBindings{} }
func (m *mockCloseableConfig) SetKeyBindings(_ types.KeyBindings) error         { return nil }
func (m *mockCloseableConfig) ResetKeyBindings() error                          { return nil }
func (m *mockCloseableConfig) GetTreeSeparator() string                         { return ":" }
func (m *mockCloseableConfig) SetTreeSeparator(_ string) error                  { return nil }
func (m *mockCloseableConfig) GetWatchInterval() time.Duration                  { return time.Second }

// mockCloseableRedis implements RedisService with a configurable Disconnect error.
type mockCloseableRedis struct {
	disconnectErr error
}

func (m *mockCloseableRedis) Disconnect() error { return m.disconnectErr }
func (m *mockCloseableRedis) Connect(_ string, _ int, _ string, _ int) error { return nil }
func (m *mockCloseableRedis) ConnectWithTLS(_ string, _ int, _ string, _ int, _ *tls.Config) error {
	return nil
}
func (m *mockCloseableRedis) ConnectCluster(_ []string, _ string) error              { return nil }
func (m *mockCloseableRedis) IsCluster() bool                                        { return false }
func (m *mockCloseableRedis) TestConnection(_ string, _ int, _ string, _ int) (time.Duration, error) {
	return 0, nil
}
func (m *mockCloseableRedis) GetTotalKeys() int64 { return 0 }
func (m *mockCloseableRedis) ScanKeys(_ string, _ uint64, _ int64) ([]types.RedisKey, uint64, error) {
	return nil, 0, nil
}
func (m *mockCloseableRedis) ScanKeysWithRegex(_ string, _ int) ([]types.RedisKey, error) {
	return nil, nil
}
func (m *mockCloseableRedis) FuzzySearchKeys(_ string, _ int) ([]types.RedisKey, error) {
	return nil, nil
}
func (m *mockCloseableRedis) GetValue(_ string) (types.RedisValue, error) {
	return types.RedisValue{}, nil
}
func (m *mockCloseableRedis) DeleteKey(_ string) error                     { return nil }
func (m *mockCloseableRedis) DeleteKeys(_ ...string) (int64, error)        { return 0, nil }
func (m *mockCloseableRedis) BulkDelete(_ string) (int, error)             { return 0, nil }
func (m *mockCloseableRedis) Rename(_, _ string) error                     { return nil }
func (m *mockCloseableRedis) Copy(_, _ string, _ bool) error               { return nil }
func (m *mockCloseableRedis) SearchByValue(_, _ string, _ int) ([]types.RedisKey, error) {
	return nil, nil
}
func (m *mockCloseableRedis) CompareKeys(_, _ string) (types.RedisValue, types.RedisValue, error) {
	return types.RedisValue{}, types.RedisValue{}, nil
}
func (m *mockCloseableRedis) GetKeyPrefixes(_ string, _ int) ([]string, error)  { return nil, nil }
func (m *mockCloseableRedis) SetString(_, _ string, _ time.Duration) error      { return nil }
func (m *mockCloseableRedis) SetTTL(_ string, _ time.Duration) error            { return nil }
func (m *mockCloseableRedis) BatchSetTTL(_ string, _ time.Duration) (int, error) { return 0, nil }
func (m *mockCloseableRedis) RPush(_ string, _ ...string) error                 { return nil }
func (m *mockCloseableRedis) LSet(_ string, _ int64, _ string) error            { return nil }
func (m *mockCloseableRedis) LRem(_ string, _ int64, _ string) error            { return nil }
func (m *mockCloseableRedis) SAdd(_ string, _ ...string) error                  { return nil }
func (m *mockCloseableRedis) SRem(_ string, _ ...string) error                  { return nil }
func (m *mockCloseableRedis) ZAdd(_ string, _ float64, _ string) error          { return nil }
func (m *mockCloseableRedis) ZRem(_ string, _ ...string) error                  { return nil }
func (m *mockCloseableRedis) HSet(_, _, _ string) error                         { return nil }
func (m *mockCloseableRedis) HDel(_ string, _ ...string) error                  { return nil }
func (m *mockCloseableRedis) XAdd(_ string, _ map[string]any) (string, error) { return "", nil }
func (m *mockCloseableRedis) XDel(_ string, _ ...string) error                  { return nil }
func (m *mockCloseableRedis) SelectDB(_ int) error                              { return nil }
func (m *mockCloseableRedis) FlushDB() error                                    { return nil }
func (m *mockCloseableRedis) GetServerInfo() (types.ServerInfo, error)           { return types.ServerInfo{}, nil }
func (m *mockCloseableRedis) GetMemoryStats() (types.MemoryStats, error) {
	return types.MemoryStats{}, nil
}
func (m *mockCloseableRedis) MemoryUsage(_ string) (int64, error) { return 0, nil }
func (m *mockCloseableRedis) SlowLogGet(_ int64) ([]types.SlowLogEntry, error) { return nil, nil }
func (m *mockCloseableRedis) ClientList() ([]types.ClientInfo, error)           { return nil, nil }
func (m *mockCloseableRedis) ClusterNodes() ([]types.ClusterNode, error)        { return nil, nil }
func (m *mockCloseableRedis) ClusterInfo() (string, error)                      { return "", nil }
func (m *mockCloseableRedis) Eval(_ string, _ []string, _ ...any) (any, error) { return nil, nil }
func (m *mockCloseableRedis) Publish(_, _ string) (int64, error)                { return 0, nil }
func (m *mockCloseableRedis) Subscribe(_ string) *redis.PubSub                  { return nil }
func (m *mockCloseableRedis) PubSubChannels(_ string) ([]string, error)         { return nil, nil }
func (m *mockCloseableRedis) SubscribeKeyspace(_ string, _ func(types.KeyspaceEvent)) error {
	return nil
}
func (m *mockCloseableRedis) UnsubscribeKeyspace() error                        { return nil }
func (m *mockCloseableRedis) ExportKeys(_ string) (map[string]any, error)     { return nil, nil }
func (m *mockCloseableRedis) ImportKeys(_ map[string]any) (int, error)        { return 0, nil }

func TestNewContainer(t *testing.T) {
	cfg := &mockCloseableConfig{}
	r := &mockCloseableRedis{}
	c := NewContainer(cfg, r)

	if c.Config != cfg {
		t.Error("Config not set correctly")
	}
	if c.Redis != r {
		t.Error("Redis not set correctly")
	}
}

func TestContainer_Close(t *testing.T) {
	t.Run("both nil no panic", func(t *testing.T) {
		c := &Container{}
		err := c.Close()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("config error only", func(t *testing.T) {
		configErr := errors.New("config close error")
		c := &Container{
			Config: &mockCloseableConfig{closeErr: configErr},
		}
		err := c.Close()
		if err != configErr {
			t.Errorf("expected config error, got %v", err)
		}
	})

	t.Run("redis error only", func(t *testing.T) {
		redisErr := errors.New("redis disconnect error")
		c := &Container{
			Redis: &mockCloseableRedis{disconnectErr: redisErr},
		}
		err := c.Close()
		if err != redisErr {
			t.Errorf("expected redis error, got %v", err)
		}
	})

	t.Run("both errors returns last", func(t *testing.T) {
		configErr := errors.New("config error")
		redisErr := errors.New("redis error")
		c := &Container{
			Config: &mockCloseableConfig{closeErr: configErr},
			Redis:  &mockCloseableRedis{disconnectErr: redisErr},
		}
		err := c.Close()
		if err != redisErr {
			t.Errorf("expected redis error (last), got %v", err)
		}
	})

	t.Run("no errors", func(t *testing.T) {
		c := &Container{
			Config: &mockCloseableConfig{},
			Redis:  &mockCloseableRedis{},
		}
		err := c.Close()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
