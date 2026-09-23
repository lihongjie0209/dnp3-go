package link

import (
	"context"
	"errors"
	"net"
	"slices"
	"testing"
	"time"
)

func TestStrictChannelRetriesConfirmedWithoutDuplicateDelivery(t *testing.T) {
	t.Parallel()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close(); _ = serverConn.Close() })
	client := NewStrictChannel(clientConn, NewStrictState(1, 10, true), 20*time.Millisecond, 2)
	server := NewStrictState(10, 1, false)
	client.State().MarkReset()
	server.MarkReset()
	serverErr := make(chan error, 1)
	go func() {
		first, err := ReadStrictFrame(serverConn)
		if err != nil {
			serverErr <- err
			return
		}
		action, err := server.Handle(first)
		if err != nil || action.Response == nil || !slices.Equal(action.UserData, []byte{1, 2, 3}) {
			serverErr <- errors.New("first frame was not delivered")
			return
		}
		second, err := ReadStrictFrame(serverConn)
		if err != nil {
			serverErr <- err
			return
		}
		duplicate, err := server.Handle(second)
		if err != nil || duplicate.Response == nil || !duplicate.Duplicate || len(duplicate.UserData) != 0 || second.Control != first.Control {
			serverErr <- errors.New("retry was not an identical duplicate")
			return
		}
		serverErr <- WriteStrictFrame(serverConn, *duplicate.Response)
	}()
	if err := client.SendConfirmed(t.Context(), []byte{1, 2, 3}, nil); err != nil {
		t.Fatal(err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestStrictChannelKeepAliveAndCancellation(t *testing.T) {
	t.Parallel()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close(); _ = serverConn.Close() })
	channel := NewStrictChannel(clientConn, NewStrictState(1, 10, true), time.Second, 0)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := channel.Reset(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}
