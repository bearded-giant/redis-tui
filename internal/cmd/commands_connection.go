package cmd

import (
	"fmt"
	"log/slog"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func (c *Commands) LoadConnections() tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionsLoadedMsg{Err: nil}
		}
		connections, err := c.config.ListConnections()
		if err != nil {
			slog.Error("Failed to load connections", "error", err)
		}
		return types.ConnectionsLoadedMsg{Connections: connections, Err: err}
	}
}

func (c *Commands) AddConnection(name, host string, port int, password string, dbNum int, useCluster bool) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionAddedMsg{Err: nil}
		}
		conn, err := c.config.AddConnection(name, host, port, password, dbNum, useCluster)
		if err != nil {
			slog.Error("Failed to add connection", "error", err)
		}
		return types.ConnectionAddedMsg{Connection: conn, Err: err}
	}
}

func (c *Commands) UpdateConnection(id int64, name, host string, port int, password string, dbNum int, useCluster bool) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionUpdatedMsg{Err: nil}
		}
		conn, err := c.config.UpdateConnection(id, name, host, port, password, dbNum, useCluster)
		if err != nil {
			slog.Error("Failed to update connection", "error", err)
		}
		return types.ConnectionUpdatedMsg{Connection: conn, Err: err}
	}
}

func (c *Commands) DeleteConnection(id int64) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionDeletedMsg{Err: nil}
		}
		err := c.config.DeleteConnection(id)
		return types.ConnectionDeletedMsg{ID: id, Err: err}
	}
}

func (c *Commands) Connect(host string, port int, password string, dbNum int, useCluster bool) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectedMsg{Err: nil}
		}
		var err error
		if useCluster {
			err = c.redis.ConnectCluster([]string{fmt.Sprintf("%s:%d", host, port)}, password)
		} else {
			err = c.redis.Connect(host, port, password, dbNum)
		}
		if err != nil {
			slog.Error("Failed to connect", "error", err)
		}
		return types.ConnectedMsg{Err: err}
	}
}

func (c *Commands) AutoConnect(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectedMsg{Err: nil}
		}
		var err error
		if conn.UseCluster {
			err = c.redis.ConnectCluster([]string{fmt.Sprintf("%s:%d", conn.Host, conn.Port)}, conn.Password)
		} else if conn.UseTLS {
			if conn.TLSConfig == nil {
				return types.ConnectedMsg{Err: fmt.Errorf("TLS requested but TLS configuration is missing")}
			}
			tlsCfg, tlsErr := conn.TLSConfig.BuildTLSConfig()
			if tlsErr != nil {
				slog.Error("Failed to build TLS config", "error", tlsErr)
				return types.ConnectedMsg{Err: tlsErr}
			}
			err = c.redis.ConnectWithTLS(conn.Host, conn.Port, conn.Password, conn.DB, tlsCfg)
		} else {
			err = c.redis.Connect(conn.Host, conn.Port, conn.Password, conn.DB)
		}
		if err != nil {
			slog.Error("Failed to connect", "error", err)
		}
		return types.ConnectedMsg{Err: err}
	}
}

func (c *Commands) Disconnect() tea.Cmd {
	return func() tea.Msg {
		if c.redis != nil {
			_ = c.redis.Disconnect()
		}
		return types.DisconnectedMsg{}
	}
}

func (c *Commands) TestConnection(host string, port int, password string, db int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectionTestMsg{Success: false, Err: nil}
		}
		latency, err := c.redis.TestConnection(host, port, password, db)
		return types.ConnectionTestMsg{Success: err == nil, Latency: latency, Err: err}
	}
}
