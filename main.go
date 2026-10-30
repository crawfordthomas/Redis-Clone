package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)
func main() {
	startServer("127.0.0.1:6379")

	
}

func startServer(address string) error {
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
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		var result []string
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Read failed: %v", err)
			break
		}
		count, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		for range count {
			arg, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Read failed: %v", err)
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
		} else if strings.ToUpper(result[0]) == "SET" {
			continue
		} else if strings.ToUpper(result[0]) == "GET" {
			continue
		} else {
			conn.Write([]byte("+PONG\r\n")) 
		}
	}
}