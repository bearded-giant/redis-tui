package cmd

import (
	"testing"

	"github.com/davidbudnick/redis-tui/internal/service"
	"github.com/davidbudnick/redis-tui/internal/testutil"
)

func newMockCmds() (*Commands, *testutil.FullMockRedisClient) {
	mock := testutil.NewFullMockRedisClient()
	cmds := NewCommands(nil, mock)
	return cmds, mock
}

func TestNewCommandsFromContainer(t *testing.T) {
	mock := testutil.NewFullMockRedisClient()
	cfg := testutil.NewTestConfig(t)
	container := &service.Container{Config: cfg, Redis: mock}
	cmds := NewCommandsFromContainer(container)
	if cmds.config != cfg {
		t.Error("config not set from container")
	}
	if cmds.redis != mock {
		t.Error("redis not set from container")
	}
}
