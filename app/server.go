package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	directory string
)

func main() {
	flag.StringVar(&directory, "directory", "/tmp", "help message")
	flag.Parse()

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}

type HTTPRequest struct {
	verb        string // GET, POST ...
	httpVersion string // HTTP/1.1
	path        string
	host        string // localhost:4221
	userAgent   string
	body        string
	encoding    string
}

func parseRequest(request string) (*HTTPRequest, error) {
	strs := strings.Split(request, "\\r\\n")

	req := HTTPRequest{}
	for _, item := range strs {
		if strings.Contains(item, "GET") || strings.Contains(item, "POST") {
			headerParts := strings.Fields(item)
			// set http verb
			req.verb = strings.Trim(headerParts[0], "\"")

			// set route
			req.path = headerParts[1]

			// set http version
			req.httpVersion = headerParts[2]
		}
		if strings.Contains(item, "Host: ") {
			req.host = item[strings.Index("Host: ", item)+len("Host: "):]
		}
		if strings.Contains(item, "User-Agent: ") {
			req.userAgent = item[strings.Index("User-Agent: ", item)+len("User-Agent: "):]
		}
		if strings.Contains(item, "Accept-Encoding: ") {
			req.encoding = item[strings.Index("Accept-Encoding: ", item)+len("Accept-Encoding: "):]
			req.encoding = strings.Trim(req.encoding, " ")
		}
	}

	if req.verb == "POST" {
		req.body = strings.TrimSuffix(strs[len(strs)-1], "\"")
	}

	return &req, nil
}

func writeResponse(conn net.Conn, res string) {
	_, err := conn.Write([]byte(res))
	if err != nil {
		fmt.Println("failed to write to connection")
		return
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading: %#v\n", err)
			}
			break
		}

		body := strconv.Quote(string(buf[:n]))
		req, _ := parseRequest(body)
		if req.path == "/" {
			writeResponse(conn, "HTTP/1.1 200 OK\r\n\r\n")
		} else if strings.Contains(req.path, "/echo/") {
			echoOut := req.path[strings.Index(req.path, "/echo/")+len("/echo/"):]
			var contEncoding = ""
			if req.encoding == "gzip" {
				fmt.Println("in encoding")
				contEncoding = fmt.Sprintf("Content-Encoding: %s\r\n", req.encoding)
			}
			fmt.Println(contEncoding)
			res := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n%sContent-Length: %d\r\n\r\n%s", contEncoding, len(echoOut), echoOut)
			writeResponse(conn, res)
		} else if strings.Contains(req.path, "/user-agent") {
			req.userAgent = strings.Trim(req.userAgent, " ")
			res := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(req.userAgent), req.userAgent)
			writeResponse(conn, res)
		} else if strings.Contains(req.path, "/files/") {
			fileName := req.path[strings.Index(req.path, "/files/")+len("/files/"):]
			filePath := fmt.Sprintf("%s%s", directory, fileName)
			fmt.Println("file path", filePath)
			if req.verb == "GET" {
				dat, err := os.ReadFile(filePath)
				if err != nil {
					fmt.Println("file not found")
					writeResponse(conn, "HTTP/1.1 404 Not Found\r\n\r\n")
					return
				}

				fmt.Print(string(dat))
				fileContents := string(dat)
				res := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(fileContents), fileContents)
				writeResponse(conn, res)
			} else {
				err := os.WriteFile(filePath, []byte(req.body), 0644)
				if err != nil {
					fmt.Println("file failed to create")
					writeResponse(conn, "HTTP/1.1 404 Not Found\r\n\r\n")
					return
				}

				writeResponse(conn, "HTTP/1.1 201 Created\r\n\r\n")
			}
		} else {
			fmt.Println("route not found")
			writeResponse(conn, "HTTP/1.1 404 Not Found\r\n\r\n")
		}
	}
}
