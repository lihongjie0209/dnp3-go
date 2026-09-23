package session

import (
	"net"
	"slices"
	"testing"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
)

func TestStrictSessionLargeApplicationOverPipe(t *testing.T) {
	t.Parallel()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close(); _ = serverConn.Close() })
	client := NewStrict(clientConn, 1, 10, true, time.Second, 1, 2048)
	server := NewStrict(serverConn, 10, 1, false, time.Second, 1, 2048)
	type result struct {
		fragment app.StrictFragment
		err      error
	}
	received := make(chan result, 1)
	go func() { fragment, err := server.ReceiveApplication(t.Context()); received <- result{fragment, err} }()
	if err := client.Reset(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	objects := make([]byte, 700)
	for index := range objects {
		objects[index] = byte(index)
	}
	want := app.StrictFragment{FIR: true, FIN: true, Sequence: 7, Function: 1, Objects: objects}
	if err := client.SendApplication(t.Context(), want, nil); err != nil {
		t.Fatal(err)
	}
	got := <-received
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.fragment.Sequence != want.Sequence || got.fragment.Function != want.Function || !slices.Equal(got.fragment.Objects, want.Objects) {
		t.Fatalf("fragment=%#v", got.fragment)
	}
}

func TestStrictSessionRejectsInvalidBound(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	session := NewStrict(client, 1, 10, true, time.Second, 0, 0)
	if err := session.SendApplicationUnconfirmed(t.Context(), app.StrictFragment{FIR: true, FIN: true, Function: 1}); err == nil {
		t.Fatal("accepted zero application bound")
	}
}

func TestStrictSessionAutomaticallyConfirmsApplicationFragment(t *testing.T) {
	t.Parallel()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close(); _ = serverConn.Close() })
	client := NewStrict(clientConn, 1, 10, true, time.Second, 0, 256)
	server := NewStrict(serverConn, 10, 1, false, time.Second, 0, 256)
	type result struct {
		fragment app.StrictFragment
		err      error
	}
	want := app.StrictFragment{FIR: true, FIN: true, CON: true, UNS: true, Sequence: 9, Function: 0x82, Objects: []byte{7}}
	confirmation := make(chan result, 1)
	go func() {
		if err := server.SendApplicationUnconfirmed(t.Context(), want); err != nil {
			confirmation <- result{err: err}
			return
		}
		fragment, err := server.ReceiveApplication(t.Context())
		confirmation <- result{fragment, err}
	}()
	received, err := client.ReceiveApplication(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if received.CON || !received.UNS || !slices.Equal(received.Objects, want.Objects) {
		t.Fatalf("received=%#v", received)
	}
	confirmed := <-confirmation
	if confirmed.err != nil {
		t.Fatal(confirmed.err)
	}
	if confirmed.fragment.Function != 0 || confirmed.fragment.Sequence != want.Sequence || !confirmed.fragment.UNS || len(confirmed.fragment.Objects) != 0 {
		t.Fatalf("confirmation=%#v", confirmed.fragment)
	}
}
