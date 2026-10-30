package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// testAddr is the loopback address the test server listens on. It is NOT the
// real Redis port (6379) so these tests don't clash with a real redis-server
// you might have running.
const testAddr = "127.0.0.1:6399"

// TestMain boots the server once for every test in this package, waits until
// it is accepting connections, then runs the tests.
//
// It calls startServer(addr string) error -- the one function the tests expect
// your implementation to provide: it binds the address and serves until the
// process exits, so we run it in the background. If you name it differently,
// change this one line to match.
func TestMain(m *testing.M) {
	go func() { _ = startServer(testAddr) }()

	if err := waitForServer(testAddr, 2*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// waitForServer polls the address until something is listening or time runs out.
func waitForServer(addr string, within time.Duration) error {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("server did not start listening on %s within %s", addr, within)
}

// sendCommandLine opens a fresh connection, writes raw bytes, and returns the
// first reply line (up to and including the trailing "\n"). It is enough for
// simple-string, error, and single-line replies; GET's bulk string will need a
// fuller reader, which we'll add when we test SET/GET.
func sendCommandLine(t *testing.T, raw string) string {
	t.Helper()
	conn, err := net.Dial("tcp", testAddr)
	if err != nil {
		t.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	return line
}

// TestPingReturnsPong: a client that sends PING must get a RESP simple-string
// PONG back.
//
//	wire out: *1\r\n$4\r\nPING\r\n   (an array of one bulk string, "PING")
//	wire in:  +PONG\r\n              (a simple string)
func TestPingReturnsPong(t *testing.T) {
	got := sendCommandLine(t, "*1\r\n$4\r\nPING\r\n")
	want := "+PONG\r\n"
	if got != want {
		t.Fatalf("PING: got %q, want %q", got, want)
	}
}
