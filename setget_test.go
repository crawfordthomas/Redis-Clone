package main

import (
	"bufio"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// getBulk sends a raw command and reads a RESP bulk-string reply.
// It returns (value, found). A null bulk string ($-1\r\n) means found == false,
// which is how Redis signals "no such key".
//
//	e.g. reply "$3\r\nbar\r\n"  -> ("bar", true)
//	     reply "$-1\r\n"        -> ("",    false)
func getBulk(t *testing.T, raw string) (string, bool) {
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

	r := bufio.NewReader(conn)
	header, err := r.ReadString('\n') // "$3\r\n" or "$-1\r\n"
	if err != nil {
		t.Fatalf("read header failed: %v", err)
	}
	if len(header) == 0 || header[0] != '$' {
		t.Fatalf("expected a bulk-string reply starting with '$', got %q", header)
	}

	n, err := strconv.Atoi(strings.TrimSpace(header[1:]))
	if err != nil {
		t.Fatalf("bad bulk length in %q: %v", header, err)
	}
	if n == -1 {
		return "", false // null bulk string: key not found
	}

	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatalf("read payload failed: %v", err)
	}
	r.ReadString('\n') // consume trailing \r\n
	return string(buf), true
}

// TestSetReturnsOK: SET stores a value and replies with the simple string OK.
//
//	wire out: *3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
//	wire in:  +OK\r\n
func TestSetReturnsOK(t *testing.T) {
	got := sendCommandLine(t, "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n")
	want := "+OK\r\n"
	if got != want {
		t.Fatalf("SET: got %q, want %q", got, want)
	}
}

// TestSetThenGet: after SET, a GET of the same key returns the stored value as
// a bulk string. Uses its own key so it doesn't collide with other tests
// sharing the one server/store.
func TestSetThenGet(t *testing.T) {
	// SET color blue
	if got := sendCommandLine(t, "*3\r\n$3\r\nSET\r\n$5\r\ncolor\r\n$4\r\nblue\r\n"); got != "+OK\r\n" {
		t.Fatalf("SET color: got %q, want +OK\\r\\n", got)
	}
	// GET color
	val, found := getBulk(t, "*2\r\n$3\r\nGET\r\n$5\r\ncolor\r\n")
	if !found {
		t.Fatalf("GET color: expected a value, got nil")
	}
	if val != "blue" {
		t.Fatalf("GET color: got %q, want %q", val, "blue")
	}
}

// TestGetMissingKey: GET of a key that was never set returns a null bulk string.
func TestGetMissingKey(t *testing.T) {
	if _, found := getBulk(t, "*2\r\n$3\r\nGET\r\n$9\r\nnosuchkey\r\n"); found {
		t.Fatalf("GET nosuchkey: expected nil (null bulk string), got a value")
	}
}
