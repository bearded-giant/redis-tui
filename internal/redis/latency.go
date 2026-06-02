package redis

import (
	"fmt"
	"strconv"

	"github.com/bearded-giant/redis-tui/internal/types"
)

// LatencyLatest runs `LATENCY LATEST` and returns one entry per tracked event.
// Each entry has Event, Timestamp, LatestMs, MaxMs.
func (c *Client) LatencyLatest() ([]types.LatencyEvent, error) {
	raw, err := c.do("LATENCY", "LATEST").Result()
	if err != nil {
		return nil, fmt.Errorf("LATENCY LATEST: %w", err)
	}
	rows, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected LATENCY LATEST shape: %T", raw)
	}
	events := make([]types.LatencyEvent, 0, len(rows))
	for _, r := range rows {
		row, ok := r.([]any)
		if !ok || len(row) < 4 {
			continue
		}
		events = append(events, types.LatencyEvent{
			Event:    asString(row[0]),
			Time:     asInt64(row[1]),
			LatestMs: asInt64(row[2]),
			MaxMs:    asInt64(row[3]),
		})
	}
	return events, nil
}

// LatencyHistory runs `LATENCY HISTORY <event>` and returns the recorded samples.
func (c *Client) LatencyHistory(event string) ([]types.LatencySample, error) {
	raw, err := c.do("LATENCY", "HISTORY", event).Result()
	if err != nil {
		return nil, fmt.Errorf("LATENCY HISTORY %s: %w", event, err)
	}
	rows, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected LATENCY HISTORY shape: %T", raw)
	}
	samples := make([]types.LatencySample, 0, len(rows))
	for _, r := range rows {
		row, ok := r.([]any)
		if !ok || len(row) < 2 {
			continue
		}
		samples = append(samples, types.LatencySample{
			Time:     asInt64(row[0]),
			LatencyMs: asInt64(row[1]),
		})
	}
	return samples, nil
}

// LatencyDoctor runs `LATENCY DOCTOR` and returns the server's narrative
// diagnosis as a single string.
func (c *Client) LatencyDoctor() (string, error) {
	out, err := c.do("LATENCY", "DOCTOR").Text()
	if err != nil {
		return "", fmt.Errorf("LATENCY DOCTOR: %w", err)
	}
	return out, nil
}

// LatencyReset clears tracked events. Pass nil/empty to reset all events.
// Returns the number of events removed.
func (c *Client) LatencyReset(events ...string) (int, error) {
	args := []any{"LATENCY", "RESET"}
	for _, e := range events {
		args = append(args, e)
	}
	n, err := c.do(args...).Int64()
	if err != nil {
		return 0, fmt.Errorf("LATENCY RESET: %w", err)
	}
	return int(n), nil
}

// LatencyMonitorThreshold reads the server's `latency-monitor-threshold`
// CONFIG value. 0 means latency monitoring is disabled and LATENCY LATEST
// will return empty until SET to >0.
func (c *Client) LatencyMonitorThreshold() (int, error) {
	res, err := c.cmdable().ConfigGet(c.ctx, "latency-monitor-threshold").Result()
	if err != nil {
		return 0, fmt.Errorf("CONFIG GET latency-monitor-threshold: %w", err)
	}
	raw, ok := res["latency-monitor-threshold"]
	if !ok {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse latency-monitor-threshold %q: %w", raw, err)
	}
	return n, nil
}

// asString tolerates string/[]byte without panic.
func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// asInt64 tolerates int64/int/string without panic.
func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case []byte:
		n, _ := strconv.ParseInt(string(x), 10, 64)
		return n
	default:
		return 0
	}
}
