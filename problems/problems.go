package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	utls "github.com/ulixee/utls"
	"golang.org/x/net/http2"
)

var (
	dialTimeout = time.Duration(15) * time.Second
)

var conn *utls.UConn

func HttpGetByHelloID(url *url.URL, helloID utls.ClientHelloID) error {
	hostname := url.Host
	addr := fmt.Sprintf("%s:%s", hostname, "443")
	config := utls.Config{ServerName: hostname}
	dialConn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		fmt.Printf("net.DialTimeout error: %s, %+v\n", addr, err)
		return err
	}
	uTlsConn := utls.UClient(dialConn, &config, helloID)
	defer uTlsConn.Close()

	err = uTlsConn.Handshake()
	if err != nil {
		fmt.Printf("uTlsConn.Handshake() %s error: %+v\n", addr, err)
		return err
	}
	fmt.Printf("#> TLS State: %+v\n", uTlsConn.ConnectionState())

	response, err := httpGetOverConn(uTlsConn, uTlsConn.ConnectionState().NegotiatedProtocol, url)
	if err != nil {
		fmt.Printf("#> %s failed: %+v\n", addr, err)
		return err
	}

	fmt.Printf("#> %s response: %+s\n", addr, dumpResponseNoBody(response))
	return nil
}

func main() {
	gstaticUrl, _ := url.Parse("https://www.gstatic.com/firebasejs/4.9.1/firebase.js")
	HttpGetByHelloID(gstaticUrl, utls.HelloChrome_72)

	ytimgUrl, _ := url.Parse("https://i.ytimg.com/vi/NfWU0Wiixuo/hqdefault.jpg")
	HttpGetByHelloID(ytimgUrl, utls.HelloChrome_72)
}

func httpGetOverConn(conn net.Conn, alpn string, url *url.URL) (*http.Response, error) {
	req := &http.Request{
		Method: "GET",
		URL:    url,
		Header: make(http.Header),
		Host:   url.Host,
	}

	switch alpn {
	case "h2":
		req.Proto = "HTTP/2.0"
		req.ProtoMajor = 2
		req.ProtoMinor = 0

		tr := http2.Transport{}
		cConn, err := tr.NewClientConn(conn)
		if err != nil {
			return nil, err
		}
		return cConn.RoundTrip(req)
	case "http/1.1", "":
		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1

		err := req.Write(conn)
		if err != nil {
			return nil, err
		}
		return http.ReadResponse(bufio.NewReader(conn), req)
	default:
		return nil, fmt.Errorf("unsupported ALPN: %v", alpn)
	}
}

func dumpResponseNoBody(response *http.Response) string {
	resp, err := httputil.DumpResponse(response, false)
	if err != nil {
		return fmt.Sprintf("failed to dump response: %v", err)
	}
	return string(resp)
}
