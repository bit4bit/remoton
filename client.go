package remoton

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"time"

	jswebsocket "github.com/gopherjs/websocket"
	"github.com/pierrec/lz4"
	"golang.org/x/net/websocket"
)

type ErrHTTP struct {
	Code int
	Msg  string
}

func (c ErrHTTP) Error() string {
	return c.Msg
}

//Client allow creation of sessions
type Client struct {
	Prefix    string
	TLSConfig *tls.Config
	Origin    string
}

//SessionClient it's a session where we can create Services a services it's
//a connection -net.Conn-
type SessionClient struct {
	*Client

	ID        string
	AuthToken string

	//WSURL web socket url by default it try
	//to guess from baseUrl
	WSURL string

	APIURL  string
	hclient *http.Client
}

//SessionListen tunnel type websocket by default
type SessionListen struct {
	*SessionClient
	service string
}

//Accept implements the net.Accept for Websocket
func (c *SessionListen) Accept() (net.Conn, error) {
	return c.dialWebsocket(c.service, "/listen")
}

//Accept implements the net.Accept for TCP
func (c *SessionListen) AcceptTCP() (net.Conn, error) {
	return c.dialTCP(c.service, "/listen")
}

func (c *SessionListen) Close() error {
	c.Destroy()
	return nil
}

func (c *SessionListen) Addr() net.Addr {
	return nil
}

//SessionListenTCP tunnel type TCP
type SessionListenTCP struct {
	*SessionClient
	service string
}

func (c *SessionListenTCP) Accept() (net.Conn, error) {
	return c.dialTCP(c.service, "/listen")
}

func (c *SessionListenTCP) Close() error {
	c.Destroy()
	return nil
}

func (c *SessionListenTCP) Addr() net.Addr {
	return nil
}

//NewSession create a session on server
func (c *Client) NewSession(_url string, authToken string) (*SessionClient, error) {

	hclient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: c.TLSConfig,
		},
	}

	req, err := http.NewRequest("POST", _url+c.Prefix+"/session", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", authToken)
	resp, err := hclient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrHTTP{resp.StatusCode, resp.Status}
	}

	session := &SessionClient{Client: c, APIURL: _url, hclient: hclient}

	if err := json.NewDecoder(resp.Body).Decode(session); err != nil {
		return nil, err
	}

	return session, nil
}

//Destroy the current session this not close active connections
func (c *SessionClient) Destroy() {
	req, _ := http.NewRequest("DELETE", c.APIURL+"/session/"+c.ID, nil)
	req.Header.Set("X-Auth-Session", c.AuthToken)
	c.hclient.Do(req)
}

//Dial create a new *service* -net.Conn- Websocket
func (c *SessionClient) Dial(service string) (net.Conn, error) {
	if runtime.GOARCH == "js" {
		return c.dialWebsocketJS(service, "/dial")
	}
	return c.dialWebsocket(service, "/dial")
}

//Dial create  a new *service* -net.Conn- TCP
func (c *SessionClient) DialTCP(service string) (net.Conn, error) {
	return c.dialTCP(service, "/dial")
}

//Listen implementes net.Listener for Websocket connections
func (c *SessionClient) Listen(service string) net.Listener {
	return &SessionListen{c, service}
}

//Listen implementes net.Listener for TCP connections
func (c *SessionClient) ListenTCP(service string) net.Listener {
	return &SessionListenTCP{c, service}
}

func (c *SessionClient) dialTCP(service string, action string) (net.Conn, error) {

	burl, err := url.Parse(c.APIURL)
	if err != nil {
		return nil, err
	}
	burl.Path += fmt.Sprintf("%s/session/%s/conn/%s%s/tcp", c.Prefix, c.ID, service, action)
	req, err := http.NewRequest("GET", burl.String(), nil)
	req.Header.Set("X-Auth-Session", c.AuthToken)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if burl.Scheme == "https" {
		conn, err = tls.Dial("tcp", burl.Host, c.TLSConfig)
	} else {
		conn, err = net.Dial("tcp", burl.Host)
	}
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	bw.WriteString("GET " + burl.RequestURI() + " HTTP/1.1\r\n")
	bw.WriteString("Host: " + burl.Host + "\r\n")
	bw.WriteString("Connection: Upgrade\r\n")
	header := http.Header{}
	header.Set("X-Auth-Session", c.AuthToken)
	err = header.Write(bw)
	if err != nil {
		return nil, err
	}
	bw.WriteString("\r\n")
	if err = bw.Flush(); err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(br, &http.Request{Method: "GET"})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sessionClient.dialTCP: http response error %s", resp.Status)
	}

	return conn, nil
}

func (c *SessionClient) dialWebsocketJS(service string, action string) (net.Conn, error) {
	var wsurl string

	if c.WSURL == "" {
		burl, err := url.Parse(c.APIURL)
		if err != nil {
			return nil, err
		}
		if burl.Scheme == "https" {
			burl.Scheme = "wss"
		} else {
			burl.Scheme = "ws"
		}

		wsurl = burl.String()
	} else {
		wsurl = c.WSURL
	}

	wsurl += fmt.Sprintf(c.Prefix+"/session/%s/conn/%s%s/websocket",
		c.ID,
		service,
		action)

	return jswebsocket.Dial(wsurl)
}

func (c *SessionClient) dialWebsocket(service string, action string) (*websocket.Conn, error) {
	var origin string
	var wsurl string
	useTls := false

	if c.Origin == "" {
		origin = "http://localhost"
	} else {
		origin = c.Origin
	}

	if c.WSURL == "" {
		burl, err := url.Parse(c.APIURL)
		if err != nil {
			return nil, err
		}
		if burl.Scheme == "https" {
			burl.Scheme = "wss"
			useTls = true
		} else {
			burl.Scheme = "ws"
		}

		wsurl = burl.String()
	} else {
		wsurl = c.WSURL
	}

	conf, err := websocket.NewConfig(wsurl, origin)
	if err != nil {
		return nil, err
	}
	conf.Protocol = []string{"binary"}
	conf.Location.Path = fmt.Sprintf(
		c.Prefix+"/session/%s/conn/%s%s/websocket", c.ID, service, action,
	)

	//TODO use root cert
	if useTls {
		conf.TlsConfig = c.TLSConfig
	}
	wsconn, err := websocket.DialConfig(conf)
	if err != nil {
		return nil, err
	}

	return wsconn, nil
}

func CompressConnection(conn net.Conn) net.Conn {
	src, dst := net.Pipe()

	go cpdeflate(src, conn)
	go cpflate(conn, src)

	return dst
}

func cpflate(dst, src io.ReadWriteCloser) {
	w := lz4.NewWriter(dst)

	errc := make(chan error)
	rbuf := make(chan []byte)

	go func() {
		for {
			buf := make([]byte, 32*512)
			rlen, err := src.Read(buf)
			if err != nil {
				errc <- err
			}
			rbuf <- buf[0:rlen]
		}
	}()

loop:
	for {
		select {
		case <-time.Tick(time.Millisecond * 16):
			w.Flush()
		case buf := <-rbuf:
			_, err := w.Write(buf)
			if err != nil {
				break loop
			}
			w.Flush()

		case <-errc:

			break loop
		}

	}
}

func cpdeflate(dst, src io.ReadWriteCloser) {
	r := lz4.NewReader(src)

	io.Copy(dst, r)
}

//NetCopy code from io.Copy but with deadline
func NetCopy(dst net.Conn, src net.Conn, deadline time.Duration) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	buf := make([]byte, 32*1024)
	for {
		src.SetReadDeadline(time.Now().Add(deadline))
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err

}
