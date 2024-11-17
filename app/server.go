package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"slices"
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
	path        string // /echo
	host        string // localhost:4221
	userAgent   string
	body        string
	encoding    string
}

type Response struct {
	version         string
	statusCode      int
	statusMessage   string
	contentType     string
	contentEncoding string
	content         string
}

const (
	OK        string = "OK"
	NOT_FOUND string = "Not Found"
	CREATED   string = "Created"
)

func (r *Response) createResponse() string {
	if r.version == "" || r.statusCode == 0 {
		fmt.Println("invalid response parameters")
		return ""
	}

	var rtnString string
	switch r.statusCode {
	case 200:
		r.statusMessage = OK
	case 404:
		r.statusMessage = NOT_FOUND
	case 201:
		r.statusMessage = CREATED
	}

	crlf := "\r\n"
	rtnString += r.version
	rtnString += fmt.Sprintf(" %d", r.statusCode)
	rtnString += fmt.Sprintf(" %s", r.statusMessage)
	rtnString += crlf

	if len(r.content) > 0 {
		contentType := fmt.Sprintf("Content-Type: %s", r.contentType)
		if r.contentEncoding != "" {
			rtnString += fmt.Sprintf("Content-Encoding: %s%s", r.contentEncoding, crlf)
		}
		contentLen := fmt.Sprintf("Content-Length: %d", len(r.content))
		rtnString += contentType
		rtnString += crlf
		rtnString += contentLen
		rtnString += fmt.Sprintf("%s%s%s", crlf, crlf, r.content)
		return rtnString
	}

	rtnString += crlf
	return rtnString
}

func parseRequest(request string) (HTTPRequest, error) {
	strs := strings.Split(request, "\\r\\n")

	var validCompressionTypes []string
	validCompressionTypes = append(validCompressionTypes, "gzip")

	req := HTTPRequest{}
	// can do this without a for loop
	for _, item := range strs {
		var x string
		if strings.Contains(item, "GET") || strings.Contains(item, "POST") {
			headerParts := strings.Fields(item)
			// set http verb
			req.verb = strings.Trim(headerParts[0], "\"")

			// set route
			req.path = headerParts[1]

			// set http version
			req.httpVersion = headerParts[2]
		}
		x = "Host: "
		if strings.Contains(item, x) {
			req.host = item[strings.Index(x, item)+len(x):]
		}
		x = "User-Agent: "
		if strings.Contains(item, x) {
			req.userAgent = item[strings.Index(x, item)+len(x):]
			req.userAgent = strings.Trim(req.userAgent, " ")
		}
		x = "Accept-Encoding: "
		if strings.Contains(item, x) {
			temp := item[strings.Index(x, item)+len(x):]
			x := strings.Split(strings.ReplaceAll(temp, " ", ""), ",")
			for _, i := range x {
				if slices.Contains(validCompressionTypes, i) {
					req.encoding = i
				}
			}
		}
	}

	if req.verb == "POST" {
		req.body = strings.TrimSuffix(strs[len(strs)-1], "\"")
	}

	return req, nil
}

func writeResponse(conn net.Conn, res *Response) {
	responseString := res.createResponse()

	_, err := conn.Write([]byte(responseString))
	if err != nil {
		fmt.Println("failed to write to connection")
		return
	}
}

func compressString(body string) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(body)); err != nil {
		log.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
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
		req, err := parseRequest(body)
		if err != nil {
			fmt.Println("error parsing request")
			return
		}

		responseWriter := Response{
			version:         req.httpVersion,
			statusCode:      200,
			contentEncoding: req.encoding,
		}

		switch {
		case req.path == "/":
			writeResponse(conn, &responseWriter)
		case strings.HasPrefix(req.path, "/echo/"):
			handleEcho(req, &responseWriter, conn)
		case strings.HasPrefix(req.path, "/user-agent"):
			handleUserAgent(req, &responseWriter, conn)
		case strings.HasPrefix(req.path, "/files/"):
			handleFile(req, &responseWriter, conn)
		default:
			responseWriter.statusCode = 404
			writeResponse(conn, &responseWriter)
		}
	}
}

func handleEcho(req HTTPRequest, r *Response, conn net.Conn) {
	echoOut := req.path[strings.Index(req.path, "/echo/")+len("/echo/"):]
	if req.encoding == "gzip" {
		echoOut = string(compressString(echoOut)[:])
	}

	r.statusCode = 200
	r.contentType = "text/plain"
	r.content = echoOut

	writeResponse(conn, r)
}

func handleFile(req HTTPRequest, r *Response, conn net.Conn) {
	fileName := req.path[strings.Index(req.path, "/files/")+len("/files/"):]
	filePath := fmt.Sprintf("%s%s", directory, fileName)

	switch req.verb {
	case "GET":
		dat, err := os.ReadFile(filePath)
		if err != nil {
			r.statusCode = 404
			writeResponse(conn, r)
			return
		}

		fileContents := string(dat)

		r.statusCode = 200
		r.contentType = "application/octet-stream"
		r.content = fileContents
		writeResponse(conn, r)

	case "POST":
		err := os.WriteFile(filePath, []byte(req.body), 0644)
		if err != nil {
			r.statusCode = 404
			writeResponse(conn, r)
			return
		}

		r.statusCode = 201
		writeResponse(conn, r)
	}
}

func handleUserAgent(req HTTPRequest, r *Response, conn net.Conn) {
	r.statusCode = 200
	r.contentType = "text/plain"
	r.content = req.userAgent
	writeResponse(conn, r)
}
