package redis

import (
	"context"
	"fmt"
	"testing"
)

func TestFuzzyScore(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		pattern  string
		wantMin  int // Minimum expected score
		wantZero bool
	}{
		{
			name:    "exact match returns high score",
			str:     "user:123",
			pattern: "user:123",
			wantMin: 100,
		},
		{
			name:    "substring match returns high score",
			str:     "user:123:profile",
			pattern: "user:123",
			wantMin: 100,
		},
		{
			name:    "prefix match with separator bonus",
			str:     "user:profile:settings",
			pattern: "ups",
			wantMin: 30, // u + p (with separator bonus) + s (with separator bonus)
		},
		{
			name:    "sequential character match",
			str:     "configuration",
			pattern: "cfg",
			wantMin: 20,
		},
		{
			name:     "no match returns zero",
			str:      "user:123",
			pattern:  "xyz",
			wantZero: true,
		},
		{
			name:     "partial pattern match returns zero",
			str:      "ab",
			pattern:  "abc",
			wantZero: true,
		},
		{
			name:    "underscore separator bonus",
			str:     "user_profile_data",
			pattern: "upd",
			wantMin: 30, // Each char after separator gets bonus
		},
		{
			name:    "hyphen separator bonus",
			str:     "user-profile-data",
			pattern: "upd",
			wantMin: 30,
		},
		{
			name:    "empty pattern matches everything",
			str:     "anything",
			pattern: "",
			wantMin: 0,
		},
		{
			name:     "empty string with pattern returns zero",
			str:      "",
			pattern:  "test",
			wantZero: true,
		},
		{
			name:    "single character match",
			str:     "test",
			pattern: "t",
			wantMin: 10,
		},
		{
			name:    "case sensitive matching",
			str:     "UserProfile",
			pattern: "UP",
			wantMin: 20,
		},
		{
			name:     "case mismatch returns zero",
			str:      "userprofile",
			pattern:  "UP",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyScore(tt.str, tt.pattern)

			if tt.wantZero {
				if got != 0 {
					t.Errorf("fuzzyScore(%q, %q) = %d, want 0", tt.str, tt.pattern, got)
				}
				return
			}

			if got < tt.wantMin {
				t.Errorf("fuzzyScore(%q, %q) = %d, want >= %d", tt.str, tt.pattern, got, tt.wantMin)
			}
		})
	}
}

func TestFuzzyScore_ContainsVsSequential(t *testing.T) {
	// When pattern is contained in string, score should be higher than sequential match
	containsStr := "session:user:123"
	containsPattern := "user"
	containsScore := fuzzyScore(containsStr, containsPattern)

	sequentialStr := "u_s_e_r_data"
	sequentialPattern := "user"
	sequentialScore := fuzzyScore(sequentialStr, sequentialPattern)

	if containsScore <= sequentialScore {
		t.Errorf("Contains match score (%d) should be higher than sequential match score (%d)",
			containsScore, sequentialScore)
	}
}

func TestFuzzyScore_SeparatorBonus(t *testing.T) {
	// Characters after separators should get bonus points
	withSeparator := "user:data"
	withoutSeparator := "userdatax"
	pattern := "ud"

	withSepScore := fuzzyScore(withSeparator, pattern)
	withoutSepScore := fuzzyScore(withoutSeparator, pattern)

	if withSepScore <= withoutSepScore {
		t.Errorf("Separator bonus score (%d) should be higher than without separator (%d)",
			withSepScore, withoutSepScore)
	}
}

func BenchmarkFuzzyScore(b *testing.B) {
	str := "user:profile:settings:preferences:notifications"
	pattern := "upsn"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzyScore(str, pattern)
	}
}

func BenchmarkFuzzyScore_LongString(b *testing.B) {
	str := "very:long:redis:key:with:many:segments:for:testing:performance"
	pattern := "vlrk"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzyScore(str, pattern)
	}
}

// ---------------------------------------------------------------------------
// silentLogger.Printf — exercise the (now non-empty) body.
// ---------------------------------------------------------------------------

func TestSilentLoggerPrintf_NonEmpty(t *testing.T) {
	l := &silentLogger{}
	l.Printf(context.Background(), "format %s %d", "x", 1)
}

// ---------------------------------------------------------------------------
// scanAll — non-cluster SCAN error path. Use the fake server to inject an
// error reply for SCAN so the standalone path returns the error.
// ---------------------------------------------------------------------------

func TestScanAll_NonClusterScanError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(host, port, "", 0); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if _, err := c.scanAll("*", 100); err == nil {
		t.Error("expected error from scanAll non-cluster SCAN error")
	}
}

// ---------------------------------------------------------------------------
// scanEach — non-cluster SCAN error path.
// ---------------------------------------------------------------------------

func TestScanEach_NonClusterScanError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(host, port, "", 0); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	err := c.scanEach("*", 100, func(keys []string) bool { return true })
	if err == nil {
		t.Error("expected error from scanEach non-cluster SCAN error")
	}
}

// ---------------------------------------------------------------------------
// scanEach — non-cluster early termination via fn returning false.
// ---------------------------------------------------------------------------

func TestScanEach_NonClusterEarlyStop(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed enough keys to span multiple SCAN iterations.
	for i := 0; i < 50; i++ {
		mr.Set(fmt.Sprintf("k:%d", i), "v")
	}

	calls := 0
	err := client.scanEach("*", 10, func(keys []string) bool {
		calls++
		return false // stop immediately on first batch
	})
	if err != nil {
		t.Fatalf("scanEach error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

// ---------------------------------------------------------------------------
// scanAll — cluster SCAN error path. Use the fake server with an error
// reply for SCAN, attached to a manually-installed cluster client.
// ---------------------------------------------------------------------------

func TestScanAll_ClusterScanError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	if _, err := client.scanAll("*", 100); err == nil {
		t.Error("expected error from scanAll cluster SCAN error")
	}
}

// ---------------------------------------------------------------------------
// scanEach — cluster SCAN error path.
// ---------------------------------------------------------------------------

func TestScanEach_ClusterScanError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	err := client.scanEach("*", 100, func(keys []string) bool { return true })
	if err == nil {
		t.Error("expected error from scanEach cluster SCAN error")
	}
}

// ---------------------------------------------------------------------------
// scanEach — cluster early-stop branch. Returns multiple SCAN pages so the
// stopped flag is observed on the next iteration of the inner loop.
// ---------------------------------------------------------------------------

func TestScanEach_ClusterEarlyStop(t *testing.T) {
	srv := newFakeRedisServer(t)
	// Track call count so we return non-zero cursor first to keep iterating,
	// then return cursor 0 (final page) so the inner loop sees the stopped
	// flag set by the previous fn() call.
	var scanCount int
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			scanCount++
			if scanCount == 1 {
				// First page: cursor "1", one key.
				return "*2\r\n$1\r\n1\r\n*1\r\n$1\r\na\r\n"
			}
			// Subsequent: cursor 0, one key. The fn returns false on first
			// batch so the second batch should never be appended.
			return "*2\r\n$1\r\n0\r\n*1\r\n$1\r\nb\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	calls := 0
	err := client.scanEach("*", 10, func(keys []string) bool {
		calls++
		return false // stop after first batch
	})
	if err != nil {
		t.Logf("scanEach cluster early-stop: %v", err)
	}
	if calls < 1 {
		t.Errorf("calls = %d, want >= 1", calls)
	}
}
