package network

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGossipSubPublishAndSubscribe tests the basic pub-sub functionality.
func TestGossipSubPublishAndSubscribe(t *testing.T) {
	t.Parallel()

	gs := NewGossipSub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go gs.Start(ctx)

	topic := "test-topic"
	msg := []byte("hello, world")

	subChan := gs.Subscribe(topic)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		receivedMsg := <-subChan
		assert.Equal(t, msg, receivedMsg, "Received message should match the scent message")
	}()

	// Allow some time for the subscriber to be ready
	time.Sleep(10 * time.Millisecond)

	err := gs.Publish(context.Background(), topic, msg)
	require.NoError(t, err, "Publish should not return an error")

	wg.Wait()
}

// TestMockP2PHostBroadcast tests the broadcast functionality of the mock host.
func TestMockP2PHostBroadcast(t *testing.T) {
	t.Parallel()

	pubsub := NewGossipSub()
	host := NewMockP2PHost(":0", pubsub)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the host and the underlying pubsub mechanism
	go func() {
		_ = host.Start(ctx)
	}()

	topic := "blocks"
	blockData := []byte("this is a block")

	subChan, err := host.Subscribe(topic)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case received := <-subChan:
			assert.Equal(t, blockData, received, "Subscriber should receive the broadcasted data")
		case <-time.After(1 * time.Second):
			t.Error("timed out waiting for message")
		}
	}()

	// Give the subscriber goroutine time to set up
	time.Sleep(50 * time.Millisecond)

	err = host.Broadcast(context.Background(), topic, blockData)
	require.NoError(t, err, "Broadcast should not fail")

	wg.Wait()
}

// TestGossipSubMultipleSubscribers ensures all subscribers to a topic receive the message.
func TestGossipSubMultipleSubscribers(t *testing.T) {
	t.Parallel()

	gs := NewGossipSub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go gs.Start(ctx)

	topic := "multi-sub"
	msg := []byte("broadcast to all")
	numSubscribers := 5

	var wg sync.WaitGroup
	wg.Add(numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		subChan := gs.Subscribe(topic)
		go func(i int, ch <-chan []byte) {
			defer wg.Done()
			select {
			case received := <-ch:
				assert.Equal(t, msg, received, "Subscriber %d received an incorrect message", i)
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Subscriber %d timed out", i)
			}
		}(i, subChan)
	}

	time.Sleep(50 * time.Millisecond) // Wait for subscriptions

	err := gs.Publish(context.Background(), topic, msg)
	require.NoError(t, err)

	wg.Wait()
}
