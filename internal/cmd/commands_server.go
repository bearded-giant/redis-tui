package cmd

import (
	"github.com/bearded-giant/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

// StartMonitor opens a MONITOR stream. Events fire via sendMsg (the model's
// tea.Program.Send wrapper). Caller holds the returned handle on the Model
// and Close()es it on screen exit. sendMsg may be nil in tests — events drop.
func (c *Commands) StartMonitor(sendMsg func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MonitorStartedMsg{}
		}
		handle, err := c.redis.StartMonitor(func(entry types.MonitorEntry) {
			if sendMsg != nil {
				sendMsg(types.MonitorEntryMsg{Entry: entry})
			}
		})
		return types.MonitorStartedMsg{Handle: handle, Err: err}
	}
}

// LoadLatencySnapshot fans out LATENCY LATEST + DOCTOR + monitor-threshold
// CONFIG GET in one Cmd, returning a combined snapshot for the screen.
func (c *Commands) LoadLatencySnapshot() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LatencySnapshotMsg{}
		}
		events, eventsErr := c.redis.LatencyLatest()
		doctor, _ := c.redis.LatencyDoctor()
		threshold, _ := c.redis.LatencyMonitorThreshold()
		return types.LatencySnapshotMsg{
			Events:    events,
			Doctor:    doctor,
			Threshold: threshold,
			Err:       eventsErr,
		}
	}
}

func (c *Commands) LoadLatencyHistory(event string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LatencyHistoryMsg{Event: event}
		}
		samples, err := c.redis.LatencyHistory(event)
		return types.LatencyHistoryMsg{Event: event, Samples: samples, Err: err}
	}
}

func (c *Commands) ResetLatency(events ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LatencyResetMsg{}
		}
		n, err := c.redis.LatencyReset(events...)
		return types.LatencyResetMsg{Count: n, Err: err}
	}
}

func (c *Commands) LoadServerInfo() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ServerInfoLoadedMsg{Err: nil}
		}
		info, err := c.redis.GetServerInfo()
		return types.ServerInfoLoadedMsg{Info: info, Err: err}
	}
}

func (c *Commands) FlushDB() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.FlushDBMsg{Err: nil}
		}
		err := c.redis.FlushDB()
		return types.FlushDBMsg{Err: err}
	}
}

func (c *Commands) GetMemoryUsage(key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MemoryUsageMsg{Key: key, Err: nil}
		}
		bytes, err := c.redis.MemoryUsage(key)
		return types.MemoryUsageMsg{Key: key, Bytes: bytes, Err: err}
	}
}

func (c *Commands) GetSlowLog(count int64) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.SlowLogLoadedMsg{Err: nil}
		}
		entries, err := c.redis.SlowLogGet(count)
		return types.SlowLogLoadedMsg{Entries: entries, Err: err}
	}
}

func (c *Commands) GetClientList() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClientListLoadedMsg{Err: nil}
		}
		clients, err := c.redis.ClientList()
		return types.ClientListLoadedMsg{Clients: clients, Err: err}
	}
}

func (c *Commands) GetMemoryStats() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MemoryStatsLoadedMsg{Err: nil}
		}
		stats, err := c.redis.GetMemoryStats()
		return types.MemoryStatsLoadedMsg{Stats: stats, Err: err}
	}
}

func (c *Commands) GetClusterInfo() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClusterInfoLoadedMsg{Err: nil}
		}
		nodes, err := c.redis.ClusterNodes()
		info, _ := c.redis.ClusterInfo()
		return types.ClusterInfoLoadedMsg{Nodes: nodes, Info: info, Err: err}
	}
}

func (c *Commands) FetchClusterNodes() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClusterNodesLoadedMsg{Err: nil}
		}
		nodes, err := c.redis.ClusterNodes()
		return types.ClusterNodesLoadedMsg{Nodes: nodes, Err: err}
	}
}

func (c *Commands) LoadLiveMetrics() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LiveMetricsMsg{Err: nil}
		}
		data, err := c.redis.GetLiveMetrics()
		return types.LiveMetricsMsg{Data: data, Err: err}
	}
}
