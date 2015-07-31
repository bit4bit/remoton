//Package remoton tunnel handlers
package remoton

import (
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
)

func init() {
	RegisterTunnelType("websocket", webSocketTunnel)
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
