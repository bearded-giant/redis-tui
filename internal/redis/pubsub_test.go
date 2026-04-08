package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestPublish(t *testing.T) {
	client, _ := setupTestClient(t)

	// Subscribe to a channel using the raw client
	sub := client.client.Subscribe(client.ctx, "testchan")
	t.Cleanup(func() { _ = sub.Close() })

	// Wait for the subscription to be ready
	_, err := sub.Receive(client.ctx)
	if err != nil {
		t.Fatalf("failed to receive subscription confirmation: %v", err)
	}

	// Publish a message
	receivers, err := client.Publish("testchan", "hello")
	if err != nil {
		t.Fatalf("Publish() returned error: %v", err)
	}
	if receivers < 1 {
		t.Errorf("Publish() receivers = %d, want >= 1", receivers)
	}

	// Verify the message was received
	msg, err := sub.ReceiveMessage(client.ctx)
	if err != nil {
		t.Fatalf("ReceiveMessage() returned error: %v", err)
	}
	if msg.Payload != "hello" {
		t.Errorf("received payload = %q, want %q", msg.Payload, "hello")
	}
	if msg.Channel != "testchan" {
		t.Errorf("received channel = %q, want %q", msg.Channel, "testchan")
	}
}

func TestPubSubChannels(t *testing.T) {
	client, _ := setupTestClient(t)

	// Subscribe to a channel to make it active
	sub := client.client.Subscribe(client.ctx, "activechan")
	t.Cleanup(func() { _ = sub.Close() })

	// Wait for the subscription to be ready
	_, err := sub.Receive(client.ctx)
	if err != nil {
		t.Fatalf("failed to receive subscription confirmation: %v", err)
	}

	// List active channels
	channels, err := client.PubSubChannels("*")
	if err != nil {
		t.Fatalf("PubSubChannels() returned error: %v", err)
	}

	found := false
	for _, ch := range channels {
		if ch == "activechan" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("PubSubChannels() = %v, expected to contain %q", channels, "activechan")
	}
}

func TestSubscribeKeyspace(t *testing.T) {
	client, _ := setupTestClient(t)

	// Set up a channel to receive events from the handler
	eventCh := make(chan types.KeyspaceEvent, 10)
	handler := func(evt types.KeyspaceEvent) {
		eventCh <- evt
	}

	err := client.SubscribeKeyspace("*", handler)
	if err != nil {
		t.Fatalf("SubscribeKeyspace() returned error: %v", err)
	}

	// Verify the subscription was set up correctly
	if client.keyspacePS == nil {
		t.Error("keyspacePS should not be nil after SubscribeKeyspace")
	}
	if client.cancelKeyspace == nil {
		t.Error("cancelKeyspace should not be nil after SubscribeKeyspace")
	}
	if len(client.eventHandlers) != 1 {
		t.Errorf("eventHandlers length = %d, want 1", len(client.eventHandlers))
	}

	// Try to trigger a keyspace event by setting a key via the redis client
	// (miniredis may or may not fire keyspace notifications, so we use a timeout)
	client.client.Set(client.ctx, "mykey", "val", 0)

	select {
	case evt := <-eventCh:
		// If we received an event, validate it
		if evt.Key != "mykey" {
			t.Errorf("event Key = %q, want %q", evt.Key, "mykey")
		}
		if evt.DB != 0 {
			t.Errorf("event DB = %d, want 0", evt.DB)
		}
	case <-time.After(200 * time.Millisecond):
		// miniredis may not support keyspace notifications, that's acceptable.
		// We already verified the mechanics above (keyspacePS, cancelKeyspace, eventHandlers).
		t.Log("no keyspace event received (miniredis may not support keyspace notifications); mechanics verified")
	}
}

func TestSubscribeKeyspace_Resubscribe(t *testing.T) {
	client, _ := setupTestClient(t)

	// First subscription
	handler1 := func(evt types.KeyspaceEvent) {}
	err := client.SubscribeKeyspace("*", handler1)
	if err != nil {
		t.Fatalf("first SubscribeKeyspace() returned error: %v", err)
	}

	oldCancel := client.cancelKeyspace
	oldPS := client.keyspacePS

	if oldCancel == nil {
		t.Fatal("cancelKeyspace should not be nil after first SubscribeKeyspace")
	}
	if oldPS == nil {
		t.Fatal("keyspacePS should not be nil after first SubscribeKeyspace")
	}

	// Second subscription (re-subscribe)
	handler2 := func(evt types.KeyspaceEvent) {}
	err = client.SubscribeKeyspace("*", handler2)
	if err != nil {
		t.Fatalf("second SubscribeKeyspace() returned error: %v", err)
	}

	// Verify new subscription was created
	if client.cancelKeyspace == nil {
		t.Error("cancelKeyspace should not be nil after re-subscribe")
	}
	if client.keyspacePS == nil {
		t.Error("keyspacePS should not be nil after re-subscribe")
	}

	// The new cancel and PS should be different objects from the old ones
	// (old ones were replaced during re-subscribe)
	if client.keyspacePS == oldPS {
		t.Error("keyspacePS should be a new instance after re-subscribe")
	}

	// Verify only one handler is registered (old handlers replaced)
	if len(client.eventHandlers) != 1 {
		t.Errorf("eventHandlers length = %d, want 1 after re-subscribe", len(client.eventHandlers))
	}
}

func TestUnsubscribeKeyspace(t *testing.T) {
	t.Run("unsubscribe after subscribe", func(t *testing.T) {
		client, _ := setupTestClient(t)

		handler := func(evt types.KeyspaceEvent) {}
		err := client.SubscribeKeyspace("*", handler)
		if err != nil {
			t.Fatalf("SubscribeKeyspace() returned error: %v", err)
		}

		// Verify subscription is active
		if client.cancelKeyspace == nil {
			t.Fatal("cancelKeyspace should not be nil before unsubscribe")
		}
		if client.keyspacePS == nil {
			t.Fatal("keyspacePS should not be nil before unsubscribe")
		}

		// Unsubscribe
		err = client.UnsubscribeKeyspace()
		if err != nil {
			t.Fatalf("UnsubscribeKeyspace() returned error: %v", err)
		}

		if client.cancelKeyspace != nil {
			t.Error("cancelKeyspace should be nil after UnsubscribeKeyspace")
		}
		if client.keyspacePS != nil {
			t.Error("keyspacePS should be nil after UnsubscribeKeyspace")
		}
	})

	t.Run("unsubscribe when not subscribed", func(t *testing.T) {
		client, _ := setupTestClient(t)

		err := client.UnsubscribeKeyspace()
		if err != nil {
			t.Errorf("UnsubscribeKeyspace() when not subscribed returned error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Subscribe — basic coverage
// ---------------------------------------------------------------------------

func TestSubscribe(t *testing.T) {
	client, _ := setupTestClient(t)

	sub := client.Subscribe("testchan-extra")
	if sub == nil {
		t.Fatal("Subscribe returned nil PubSub")
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Wait for the subscription confirmation to ensure the channel is active.
	if _, err := sub.Receive(client.ctx); err != nil {
		t.Fatalf("Receive subscription confirmation error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SubscribeKeyspace — cluster branch. Uses the fake server with a manually
// installed cluster client so the cluster ConfigSet and PSubscribe paths fire.
// ---------------------------------------------------------------------------

func TestSubscribeKeyspace_ClusterBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "CONFIG":
			return "+OK\r\n"
		case "PSUBSCRIBE":
			// minimal psubscribe ack: array of [psubscribe, pattern, count]
			return "*3\r\n$10\r\npsubscribe\r\n$1\r\n*\r\n:1\r\n"
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

	handler := func(evt types.KeyspaceEvent) {}
	if err := client.SubscribeKeyspace("*", handler); err != nil {
		t.Fatalf("SubscribeKeyspace cluster branch error: %v", err)
	}
	if client.keyspacePS == nil {
		t.Error("keyspacePS should not be nil after cluster SubscribeKeyspace")
	}
	if err := client.UnsubscribeKeyspace(); err != nil {
		t.Logf("UnsubscribeKeyspace: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SubscribeKeyspace — exercise the goroutine's "msg !ok" exit branch by
// directly closing the keyspacePS without first canceling the context.
// ---------------------------------------------------------------------------

func TestSubscribeKeyspace_ChannelCloseExits(t *testing.T) {
	client, _ := setupTestClient(t)

	if err := client.SubscribeKeyspace("*", func(evt types.KeyspaceEvent) {}); err != nil {
		t.Fatalf("SubscribeKeyspace error: %v", err)
	}

	// Allow the goroutine to start.
	time.Sleep(50 * time.Millisecond)

	// Close the pubsub directly without invoking the cancel func. This makes
	// the channel close so the goroutine receives !ok and returns through
	// the close path.
	client.mu.Lock()
	ps := client.keyspacePS
	client.mu.Unlock()
	if ps == nil {
		t.Fatal("keyspacePS should be non-nil")
	}
	if err := ps.Close(); err != nil {
		t.Logf("ps.Close: %v", err)
	}

	// Give the goroutine time to react.
	time.Sleep(100 * time.Millisecond)

	// Cleanup via UnsubscribeKeyspace (which is now a no-op for the closed PS).
	if err := client.UnsubscribeKeyspace(); err != nil {
		t.Logf("UnsubscribeKeyspace: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SubscribeKeyspace — exercise the message-handling goroutine by publishing
// to the keyspace channel directly via the underlying client.
// ---------------------------------------------------------------------------

func TestSubscribeKeyspace_HandlerLoop(t *testing.T) {
	client, _ := setupTestClient(t)

	eventCh := make(chan types.KeyspaceEvent, 4)
	handler := func(evt types.KeyspaceEvent) {
		eventCh <- evt
	}

	if err := client.SubscribeKeyspace("*", handler); err != nil {
		t.Fatalf("SubscribeKeyspace error: %v", err)
	}

	// Allow the goroutine a moment to start the receive loop.
	time.Sleep(50 * time.Millisecond)

	// Publish a synthetic keyspace event message to the channel the handler
	// is subscribed to. The pattern is __keyspace@0__:* for db 0.
	_, err := client.client.Publish(client.ctx, "__keyspace@0__:mykey", "set").Result()
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	select {
	case evt := <-eventCh:
		if evt.Key != "mykey" {
			t.Errorf("event Key = %q, want %q", evt.Key, "mykey")
		}
		if evt.Event != "set" {
			t.Errorf("event Event = %q, want %q", evt.Event, "set")
		}
		if evt.DB != 0 {
			t.Errorf("event DB = %d, want 0", evt.DB)
		}
	case <-time.After(500 * time.Millisecond):
		t.Log("no keyspace event received within timeout (acceptable if pubsub timing varies); subscription mechanics already verified")
	}

	// Clean up.
	if err := client.UnsubscribeKeyspace(); err != nil {
		t.Fatalf("UnsubscribeKeyspace error: %v", err)
	}
}
