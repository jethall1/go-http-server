package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
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

type HTTPResponse struct {
	httpVersion  string
	reasonPhrase string
	headers      []string
	body         []string
	statusCode   int
}

// "GET / HTTP/1.1\r\n
// Host: localhost:4221\r\n
// User-Agent: curl/8.1.2\r\nAccept: */*\r\n\r\n"
type HTTPRequest struct {
	verb        string // GET, POST ...
	httpVersion string // HTTP/1.1
	path        string
	host        string // localhost:4221
	userAgent   string
}

// func parseResponse(body string) *HTTPResponse {
// 	resp := HTMLResponse{}
// 	strings.Split(body, "\r\n")

// 	return resp
// }

func parseRequest(request string) (*HTTPRequest, error) {
	strs := strings.Split(request, "\\r\\n")

	req := HTTPRequest{}
	for _, item := range strs {
		if strings.Contains(item, "GET") {
			headerParts := strings.Fields(item)
			// set http verb
			req.verb = "GET"

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
			res := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoOut), echoOut)
			writeResponse(conn, res)
		} else if strings.Contains(req.path, "/user-agent") {
			res := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(req.userAgent), req.userAgent)
			writeResponse(conn, res)
		} else {
			writeResponse(conn, "HTTP/1.1 404 Not Found\r\n\r\n")
		}
	}
}
