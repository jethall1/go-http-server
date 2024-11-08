package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

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
	host        string // localhost:4221
	headers     string // user-agent...
}

// func parseResponse(body string) *HTTPResponse {
// 	resp := HTMLResponse{}
// 	strings.Split(body, "\r\n")

// 	return resp
// }

// func parseRequest(request string) (*HTTPRequest, error) {
// 	strs := strings.Split(request, "\r\n")
// 	if len(strs) < 3 {
// 		fmt.Println("request invalid")
// 		return nil, errors.New("too few values")
// 	}

// 	req := HTTPRequest{}

// 	req.headers = strs[2]

// 	return req
// }

func handleRequest(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading: %#v\n", err)
			}
			break
		}

		// body := strconv.Quote(string(buf[:n]))
		// expectedReq := strconv.Quote("GET / HTTP/1.1\r\nHost: localhost:4221")
		// fmt.Printf("Recieved: %s\n", body)
		// fmt.Printf("expected: %s\n", expectedReq)
		// if body == expectedReq {
		// fmt.Println("req does equal that, responding...")
		_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		if err != nil {
			fmt.Println("failed to write to connection")
			return
		}
	}
}
