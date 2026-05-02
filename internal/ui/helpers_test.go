package ui

import (
	"testing"

	"github.com/bearded-giant/redis-tui/internal/cmd"
	"github.com/bearded-giant/redis-tui/internal/testutil"
)

// newTestModel returns a fully-wired Model backed by mock redis + config services
// via a live *cmd.Commands. Every message/screen handler can assume m.Cmds is non-nil.
func newTestModel(t *testing.T) (Model, *testutil.FullMockRedisClient, *testutil.MockConfigClient) {
	t.Helper()
	redis := testutil.NewFullMockRedisClient()
	cfg := testutil.NewMockConfigClient()
	m := NewModel()
	m.Cmds = cmd.NewCommands(cfg, redis)
	m.Width = 120
	m.Height = 40
	m.ScanSize = 100
	return m, redis, cfg
}
