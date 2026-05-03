package service

import (
	"errors"
	"testing"

	"github.com/bearded-giant/redis-tui/internal/testutil"
)

func newMockConfig(closeErr error) *testutil.MockConfigClient {
	m := testutil.NewMockConfigClient()
	m.CloseError = closeErr
	return m
}

func newMockRedis(disconnectErr error) *testutil.FullMockRedisClient {
	m := testutil.NewFullMockRedisClient()
	m.MockRedisClient.DisconnectError = disconnectErr
	return m
}

func TestNewContainer(t *testing.T) {
	cfg := newMockConfig(nil)
	r := newMockRedis(nil)
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
			Config: newMockConfig(configErr),
		}
		err := c.Close()
		if err != configErr {
			t.Errorf("expected config error, got %v", err)
		}
	})

	t.Run("redis error only", func(t *testing.T) {
		redisErr := errors.New("redis disconnect error")
		c := &Container{
			Redis: newMockRedis(redisErr),
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
			Config: newMockConfig(configErr),
			Redis:  newMockRedis(redisErr),
		}
		err := c.Close()
		if err != redisErr {
			t.Errorf("expected redis error (last), got %v", err)
		}
	})

	t.Run("no errors", func(t *testing.T) {
		c := &Container{
			Config: newMockConfig(nil),
			Redis:  newMockRedis(nil),
		}
		err := c.Close()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
