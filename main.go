package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Store struct {
	mu sync.Mutex
	data map[string][]byte
}

func NewStore() *Store {
	return &Store{
		data: make(map[string][]byte),
	}
}

func (s *Store) Set(key string, val []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

func (s *Store) Get(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *Store) Del(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[key]
	delete(s.data, key)
	return ok
}

func main() {
	serve()
}

func serve() {
	fmt.Println("Mini-Redis 1.0")
	listener, err := net.Listen("tcp", ":8379")

	if err != nil {
		log.Fatal("Error listening:", err)
	}

	defer listener.Close()

	store := NewStore()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting conn:", err)
			continue
		}
		fmt.Println("Connected")
		go handleConnection(conn, store)
	}
}

func handleConnection(conn net.Conn, store *Store) {
	for {
		reader := bufio.NewReader(conn)

		command, err := parseCommand(reader)

		// when the client exits, err is EOF
		if err != nil {
			conn.Close()
			log.Printf("Disconnected")
			return
		}

		log.Printf("COMMAND: %v\n", command)

		switch strings.ToUpper(command[0]) {
		case "PING":
			response := []byte("+PONG\r\n")
			_, err = conn.Write(response)
		case "COMMAND":
			// redis-cli sends COMMAND DOCS when first connecting
			// and I'm too lazy to implement it correctly
			response := []byte("*0\r\n")
			_, err = conn.Write(response)
		case "SET":
			if len(command) < 3 {
				_, err = conn.Write([]byte("-Error wrong number of arguments for 'set' command\r\n"))
				continue
			}
			key := command[1]
			val := command[2]
			store.Set(key, []byte(val))
			_, err = conn.Write([]byte("+OK\r\n"))
		case "GET":
			if len(command) < 2 {
				_, err = conn.Write([]byte("-Error wrong number of arguments for 'get' command\r\n"))
				continue
			}
			key := command[1]
			val, ok := store.Get(key)
			if ok {
				response, err := buildBulkString(string(val))
				if err != nil {
					_, err = conn.Write([]byte("-ERR Server error.\r\n"))
				} else {
					_, err = conn.Write(response)
				}
			} else {
				// RESP2 bulk nil
				// https://redis.io/docs/latest/develop/reference/protocol-spec/#null-bulk-strings
				_, err = conn.Write([]byte("$-1\r\n"))
			}
		case "DEL":
			if len(command) < 2 {
				_, err = conn.Write([]byte("-Error wrong number of arguments for 'del' command\r\n"))
				continue
			}
			key := command[1]
			ok := store.Del(key)
			if ok {
				_, err = conn.Write([]byte(":1\r\n"))
			} else {
				_, err = conn.Write([]byte(":0\r\n"))
			}
		}

		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}
}

func parseCommand(r *bufio.Reader) ([]string, error) {
	command := make([]string, 0)

	s, err := r.ReadString('\n')

	if err != nil {
		return nil, err
	}

	// commands are encoded in RESP as an array of bulk strings
	// *<number>\r\n<element-1>...<element-n>
	
	if len(s) < 4 || s[0] != '*' {
		return nil, fmt.Errorf("Error: Expecting RESP array of bulk strings.")
	}

	length, err := strconv.Atoi(s[1:len(s)-2])

	if err != nil {
		return nil, fmt.Errorf("Invalid command length: %v", err)
	}

	// a bulk string in RESP:
	// $<number>\r\n<string>\r\n

	for range length {
		_, err := r.ReadString('\n')

		if err != nil {
			return nil, err
		}

		s, err = r.ReadString('\n')

		if err != nil {
			return nil, err
		}

		command = append(command, s[:len(s)-2])
	}

	return command, nil
}

func buildBulkStringArray(ss []string) ([]byte, error) {
	length := strconv.Itoa(len(ss))

	array := make([]byte, 0, 3 + len(length))

	array = append(array, '*')
	array = append(array, length...)
	array = append(array, '\r')
	array = append(array, '\n')

	for i := range ss {
		s, err := buildBulkString(ss[i])
		if err != nil {
			return array, err
		}
		array = append(array, s...)
	}

	return array, nil
}

func buildBulkString(s string) ([]byte, error) {
	length := strconv.Itoa(len(s))

	b := make([]byte, 0, 5 + len(length) + len(s))

	b = append(b, '$')
	b = append(b, length...)
	b = append(b, '\r')
	b = append(b, '\n')
	b = append(b, s...)
	b = append(b, '\r')
	b = append(b, '\n')

	return b, nil
}

