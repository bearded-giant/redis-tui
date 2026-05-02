package cmd

import (
	"github.com/bearded-giant/redis-tui/internal/service"
)

// Commands provides tea.Cmd factories with injected dependencies.
// Use this struct instead of global functions for better testability.
type Commands struct {
	config service.ConfigService
	redis  service.RedisService
}

// NewCommands creates a new Commands instance with the provided services.
func NewCommands(config service.ConfigService, redis service.RedisService) *Commands {
	return &Commands{
		config: config,
		redis:  redis,
	}
}

// NewCommandsFromContainer creates a new Commands instance from a service container.
func NewCommandsFromContainer(c *service.Container) *Commands {
	return &Commands{
		config: c.Config,
		redis:  c.Redis,
	}
}
