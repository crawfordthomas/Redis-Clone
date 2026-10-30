package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Store struct {
	mu sync.RWMutex
	data map[string]string
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data[key]
	return val, exists
}

func main() {
	startServer("127.0.0.1:6379")

}

func startServer(address string) error {
	store := NewStore()
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn, store)
	}
}

func handleConnection(conn net.Conn, store *Store) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		var result []string
		line, err := reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Read error: %v", err)
			}
			break
		}
		count, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		for range count {
			arg, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Truncated command: %v", err)
				break
			}
			size, err := strconv.Atoi(strings.TrimSpace(arg[1:]))
			buf := make([]byte, size)
			io.ReadFull(reader, buf)
			result = append(result, string(buf))
			reader.ReadString('\n')
		}
		
		if len(result) == 0 {
			continue
		}

		switch strings.ToUpper(result[0]) {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "SET":
			if len(result) != 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
				continue
			}
			store.Set(result[1], result[2])
			conn.Write([]byte("+OK\r\n"))
		case "GET":
			if len(result) != 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				continue
			}
			value, ok := store.Get(result[1])
			if ok {
				len := strconv.Itoa(len(value))
				conn.Write([]byte("$" + len + "\r\n" + value + "\r\n"))
			} else {
				conn.Write([]byte("$-1\r\n"))
			}
		default:
			conn.Write([]byte("-ERR unknown command '" + result[0] + "'\r\n"))
		}
	}
}