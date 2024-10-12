package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var directory string

func main() {
	// Parse command-line flags
	flag.StringVar(&directory, "directory", "", "the directory to serve files from")
	flag.Parse()

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

	// Read the request
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from connection:", err.Error())
		return
	}

	// Parse the request
	request := string(buffer[:n])
	lines := strings.Split(request, "\r\n")
	if len(lines) < 1 {
		fmt.Println("Invalid request format")
		return
	}

	// Parse the request line
	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 2 {
		fmt.Println("Invalid request line format")
		return
	}

	path := requestLine[1]

	// Parse headers
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

	// Handle different paths
	if path == "/" {
		sendResponse(conn, "200 OK", "", "")
	} else if strings.HasPrefix(path, "/echo/") {
		echoStr := strings.TrimPrefix(path, "/echo/")
		sendResponse(conn, "200 OK", "text/plain", echoStr)
	} else if path == "/user-agent" {
		userAgent := headers["user-agent"]
		sendResponse(conn, "200 OK", "text/plain", userAgent)
	} else if strings.HasPrefix(path, "/files/") {
		if directory == "" {
			sendResponse(conn, "404 Not Found", "", "")
		} else {
			filename := strings.TrimPrefix(path, "/files/")
			handleFileRequest(conn, filename)
		}
	} else {
		sendResponse(conn, "404 Not Found", "", "")
	}
}

func handleFileRequest(conn net.Conn, filename string) {
	filePath := filepath.Join(directory, filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			sendResponse(conn, "404 Not Found", "", "")
		} else {
			fmt.Println("Error reading file:", err.Error())
			sendResponse(conn, "500 Internal Server Error", "", "")
		}
		return
	}

	sendResponse(conn, "200 OK", "application/octet-stream", string(content))
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
