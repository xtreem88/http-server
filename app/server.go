package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server listening on port 4221")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from connection:", err.Error())
		return
	}

	request := string(buffer[:n])
	lines := strings.Split(request, "\r\n")
	if len(lines) < 1 {
		fmt.Println("Invalid request format")
		return
	}

	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 2 {
		fmt.Println("Invalid request line format")
		return
	}

	path := requestLine[1]

	headers := make(map[string]string)
	for _, line := range lines[1:] {
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(parts[0])] = parts[1]
		}
	}

	if path == "/" {
		sendResponse(conn, "200 OK", "", "")
	} else if strings.HasPrefix(path, "/echo/") {
		echoStr := strings.TrimPrefix(path, "/echo/")
		sendResponse(conn, "200 OK", "text/plain", echoStr)
	} else if path == "/user-agent" {
		userAgent := headers["user-agent"]
		sendResponse(conn, "200 OK", "text/plain", userAgent)
	} else {
		sendResponse(conn, "404 Not Found", "", "")
	}
}

func sendResponse(conn net.Conn, status, contentType, body string) {
	response := fmt.Sprintf("HTTP/1.1 %s\r\n", status)

	if contentType != "" {
		response += fmt.Sprintf("Content-Type: %s\r\n", contentType)
		response += fmt.Sprintf("Content-Length: %d\r\n", len(body))
	}

	response += "\r\n" + body

	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing to connection:", err.Error())
	}
}
