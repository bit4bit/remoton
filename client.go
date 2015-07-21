package remoton

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/pierrec/lz4"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type ErrHTTP struct {
	Code int
	Msg  string
}

func (c ErrHTTP) Error() string {
	return c.Msg
}

type Client struct {
	Prefix    string
	TLSConfig *tls.Config
	Origin    string
}

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

type SessionListen struct {
	*SessionClient
	service string
}

func (c *Client) Session(id string, authToken string) *SessionClient {
	return &SessionClient{Client: c, ID: id, AuthToken: authToken}
}

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

func (c *SessionClient) Destroy() {
	req, _ := http.NewRequest("DELETE", c.APIURL+"/session/"+c.ID, nil)
	req.Header.Set("X-Auth-Session", c.AuthToken)
	c.hclient.Do(req)
}

func (c *SessionClient) Dial(service string) (net.Conn, error) {
	return c.dial(service, "/dial")
}

func (c *SessionClient) Listen(service string) *SessionListen {
	return &SessionListen{c, service}
}

func (c *SessionListen) Accept() (net.Conn, error) {
	return c.dial(c.service, "/listen")
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

func (c *SessionListen) Close() error {
	c.Destroy()
	return nil
}

func (c *SessionListen) Addr() net.Addr {
	return nil
}

func (c *SessionClient) dial(service string, action string) (*websocket.Conn, error) {
	var origin string
	var wsurl string

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

		burl.Scheme = "wss"
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
		c.Prefix+"/session/%s/conn/%s%s", c.ID, service, action,
	)

	//TODO use root cert
	conf.TlsConfig = c.TLSConfig
	conf.Header.Set("X-Auth-Session", c.AuthToken)

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
