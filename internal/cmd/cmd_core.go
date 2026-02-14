package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/db"
	"github.com/davidbudnick/redis-tui/internal/redis"
)

var (
	Config       *db.Config
	RedisClient  *redis.Client
	ScanSize     int64 = 1000
	IncludeTypes bool  = true
	Version      string
)
