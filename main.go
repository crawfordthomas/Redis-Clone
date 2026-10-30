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
	"time"
)

type Entry struct {
	value string
	expiresAt time.Time
}

type Store struct {
	mu sync.RWMutex
	data map[string]Entry
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

func (s *Store) Set(key, value string, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Entry{value: value, expiresAt: expiresAt}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	entry, exists := s.data[key]
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		s.mu.RUnlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		entry, exists := s.data[key]
		if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
			delete(s.data, key)
			return "", false
		}
		return entry.value, exists
	}
	defer s.mu.RUnlock()
	return entry.value, exists
}

func (s *Store) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *Store) startSweeper(interval time.Duration) {
	batchSize := 20
	threshold := 10
	for {
		removed := s.sweepBatch(batchSize)
		if removed > threshold { continue }
		time.Sleep(interval)
	}
}

func (s *Store) sweepBatch(batchSize int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	checked := 0
	removed := 0
	for key, entry := range s.data {
		if checked >= batchSize { break }
		checked++
		if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
			delete(s.data, key)
			removed++
		}
	}
	return removed
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
	go store.startSweeper(5 * time.Minute)

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
			expiresAt := time.Time{}
			if len(result) != 3 {
				if len(result) == 5 && strings.ToUpper(result[3]) == "EX" {
					seconds, err := strconv.Atoi(result[4])
					if err != nil {
						conn.Write([]byte("-ERR value is not an integer"))
						continue
					}
					expiresAt = time.Now().Add(time.Duration(seconds) * time.Second)
				} else {
					conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
					continue
				}
			}
			store.Set(result[1], result[2], expiresAt)
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