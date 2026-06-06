package cmd

import (
	"github.com/bearded-giant/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func (c *Commands) SearchByValue(pattern, valueSearch string, maxKeys int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeysLoadedMsg{Err: nil}
		}
		keys, err := c.redis.SearchByValue(pattern, valueSearch, maxKeys)
		return types.KeysLoadedMsg{Keys: keys, Cursor: 0, Err: err}
	}
}

func (c *Commands) RegexSearch(pattern string, maxKeys int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.RegexSearchResultMsg{Err: nil}
		}
		keys, err := c.redis.ScanKeysWithRegex(pattern, maxKeys)
		return types.RegexSearchResultMsg{Keys: keys, Err: err}
	}
}

func (c *Commands) FuzzySearch(term string, maxKeys int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.FuzzySearchResultMsg{Err: nil}
		}
		keys, err := c.redis.FuzzySearchKeys(term, maxKeys)
		return types.FuzzySearchResultMsg{Keys: keys, Err: err}
	}
}

// CountMatches scans the full keyspace for pattern and returns the total count.
// Intermediate progress is emitted via sendMsg (cap ~10 updates/sec to avoid
// flooding the program). Final count returned as MatchCountProgressMsg{Done:true}.
// seq is checked against the active SearchSeq on receipt; stale msgs dropped.
func (c *Commands) CountMatches(pattern string, maxKeys, seq int, sendMsg func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MatchCountProgressMsg{Seq: seq, Done: true}
		}
		var lastSent uint64
		count, stopped, err := c.redis.CountMatches(pattern, maxKeys, func(running uint64) bool {
			if sendMsg != nil && running-lastSent >= 500 {
				lastSent = running
				sendMsg(types.MatchCountProgressMsg{Seq: seq, Count: running})
			}
			return true
		})
		return types.MatchCountProgressMsg{Seq: seq, Count: count, Stopped: stopped, Done: true, Err: err}
	}
}

func (c *Commands) CompareKeys(key1, key2 string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.CompareKeysResultMsg{Err: nil}
		}
		val1, val2, err := c.redis.CompareKeys(key1, key2)
		return types.CompareKeysResultMsg{Key1Value: val1, Key2Value: val2, Err: err}
	}
}

func (c *Commands) LoadKeyPrefixes(separator string, maxDepth int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.TreeNodeExpandedMsg{Err: nil}
		}
		prefixes, err := c.redis.GetKeyPrefixes(separator, maxDepth)
		return types.TreeNodeExpandedMsg{Children: prefixes, Err: err}
	}
}
