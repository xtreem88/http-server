package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var directory string

func main() {
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
	if len(requestLine) < 3 {
		fmt.Println("Invalid request line format")
		return
	}

	method := requestLine[0]
	path := requestLine[1]

	headers := make(map[string]string)
	var bodyStart int
	for i, line := range lines[1:] {
		if line == "" {
			bodyStart = i + 2
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(parts[0])] = parts[1]
		}
	}

	if method == "GET" {
		handleGetRequest(conn, path, headers)
	} else if method == "POST" {
		handlePostRequest(conn, path, headers, strings.Join(lines[bodyStart:], "\r\n"))
	} else {
		sendResponse(conn, "405 Method Not Allowed", "", "")
	}
}

func handleGetRequest(conn net.Conn, path string, headers map[string]string) {
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

func handlePostRequest(conn net.Conn, path string, headers map[string]string, body string) {
	if !strings.HasPrefix(path, "/files/") || directory == "" {
		sendResponse(conn, "404 Not Found", "", "")
		return
	}

	filename := strings.TrimPrefix(path, "/files/")
	filePath := filepath.Join(directory, filename)

	contentLength, err := strconv.Atoi(headers["content-length"])
	if err != nil {
		fmt.Println("Invalid Content-Length:", err.Error())
		sendResponse(conn, "400 Bad Request", "", "")
		return
	}

	if len(body) < contentLength {
		remainingBytes := contentLength - len(body)
		additionalBuffer := make([]byte, remainingBytes)
		_, err := conn.Read(additionalBuffer)
		if err != nil {
			fmt.Println("Error reading additional data:", err.Error())
			sendResponse(conn, "500 Internal Server Error", "", "")
			return
		}
		body += string(additionalBuffer)
	}

	err = ioutil.WriteFile(filePath, []byte(body), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err.Error())
		sendResponse(conn, "500 Internal Server Error", "", "")
		return
	}

	sendResponse(conn, "201 Created", "", "")
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
