//Package remoton tunnel handlers
package remoton

import (
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
)

func init() {
	RegisterTunnelType("websocket", webSocketTunnel)
	RegisterTunnelType("tcp", tcpTunnel)
}

func webSocketTunnel(src net.Conn) http.Handler {

	handshake := func(conf *websocket.Config, r *http.Request) error {
		conf.Protocol = []string{"binary"}
		return nil
	}

	handler := websocket.Handler(func(ws *websocket.Conn) {
		ws.PayloadType = websocket.BinaryFrame
		<-pipe(src, ws)
		ws.Close()
		src.Close()
	})

	ws := websocket.Server{
		Handshake: handshake,
		Handler:   handler,
	}

	return ws
}

type tcpTunnelHandler struct {
	endpoint net.Conn
}

func (c *tcpTunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}

	defer conn.Close()
	fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\n")
	buf.WriteString("\r\n")
	buf.Flush()

	if conn == nil {
		panic("unexpected nil conn")
	}
	<-pipe(c.endpoint, conn)
}

func tcpTunnel(src net.Conn) http.Handler {
	return &tcpTunnelHandler{src}
}

func pipe(dst io.ReadWriteCloser, src io.ReadWriteCloser) chan error {
	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(dst, src)
		errc <- err
	}()

	go func() {
		_, err := io.Copy(src, dst)
		errc <- err
	}()

	return errc
}
