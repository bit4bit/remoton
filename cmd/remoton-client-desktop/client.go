package main

import (
	"crypto/x509"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bit4bit/remoton"
	"github.com/bit4bit/remoton/xpra"
)

type callbackNewConnection func(net.Addr)

//chatRemoton handle remote chat
type chatRemoton struct {
	cbSend map[net.Conn]func(string)
	onRecv func(msg string)
}

func newChatRemoton() *chatRemoton {
	return &chatRemoton{
		cbSend: make(map[net.Conn]func(string)),
	}
}

func (c *chatRemoton) Send(msg string) {
	for _, f := range c.cbSend {
		if f != nil {
			f(msg)
		}
	}

}

func (c *chatRemoton) OnRecv(f func(msg string)) {
	c.onRecv = f
}
func (c *chatRemoton) init() {
	if c.cbSend == nil {
		c.cbSend = make(map[net.Conn]func(string))
	}
}

func (c *chatRemoton) Start(session *remoton.SessionClient) {
	go c.start(session)
}

func (c *chatRemoton) start(session *remoton.SessionClient) {

	l := session.Listen("chat")

	for {
		wsconn, err := l.Accept()
		if err != nil {
			break
		}

		c.init()

		go func(remoteConn net.Conn) {
			c.cbSend[remoteConn] = func(msg string) {
				remoteConn.Write([]byte(msg))
			}

			for {
				buf := make([]byte, 32*512)
				rlen, err := remoteConn.Read(buf)
				if err != nil {
					delete(c.cbSend, remoteConn)
					break
				}
				if c.onRecv != nil {
					c.onRecv(strings.TrimSpace(string(buf[0:rlen])))
				}
			}

		}(wsconn)
	}

}

func (c *chatRemoton) Stop() {
}

type vncRemoton struct {
	conn         net.Conn
	onConnection func(net.Addr)
}

func newVncRemoton() *vncRemoton {
	return &vncRemoton{}
}

func (c *vncRemoton) Start(session *remoton.SessionClient) error {
	var err error
	port, _ := c.findFreePort()
	addrSrv := "localhost:" + port
	err = xpra.Bind(addrSrv)
	if err != nil {
		log.Error("vncRemoton:", err)
		return err
	}

	conn, err := net.DialTimeout("tcp", addrSrv, time.Second*3)
	if err != nil {
		xpra.Terminate()
		return err
	}
	conn.Close()

	go c.start(session, addrSrv)
	return nil
}

func (c *vncRemoton) start(session *remoton.SessionClient, addrSrv string) {
	l := session.Listen("nx")
	for {
		log.Println("vncRemoton.start: waiting connection")
		wsconn, err := l.Accept()
		if err != nil {
			log.Error(err)
			break
		}

		if c.onConnection != nil {
			c.onConnection(wsconn.RemoteAddr())
		}

		conn, err := net.Dial("tcp", addrSrv)
		if err != nil {
			log.Error("vncRemoton.start:", addrSrv, err)
			break
		}

		go c.handleTunnel(conn, wsconn)
	}
}

func (c *vncRemoton) handleTunnel(local net.Conn, remote net.Conn) {
	log.Println("vncRemoton.handleTunnel")
	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(local, remote)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(remote, local)
		errc <- err
	}()

	log.Println("vncRemoton: closing connections", <-errc)
}

func (c *vncRemoton) findFreePort() (string, int) {
	startPort := 5900

	for ; startPort < 65534; startPort++ {
		conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(startPort))
		if err != nil {
			return strconv.Itoa(startPort), startPort
		}
		conn.Close()
	}
	return "", -1
}

func (c *vncRemoton) OnConnection(cb func(addr net.Addr)) {
	c.onConnection = cb
}

func (c *vncRemoton) Stop() {
	if c.conn != nil {
		c.conn.Close()
	}

	xpra.Terminate()
}

type clientRemoton struct {
	client  *remoton.Client
	Chat    *chatRemoton
	VNC     *vncRemoton
	session *remoton.SessionClient
	started bool
}

func newClient(rclient *remoton.Client) *clientRemoton {

	return &clientRemoton{client: rclient,
		Chat:    newChatRemoton(),
		VNC:     newVncRemoton(),
		started: false}
}

func (c *clientRemoton) Started() bool {
	return c.started
}

func (c *clientRemoton) SetCertPool(roots *x509.CertPool) {
	c.client.TLSConfig.RootCAs = roots
}

func (c *clientRemoton) Start(srvAddr string, authToken string) error {
	var err error
	c.session, err = c.client.NewSession("https://"+srvAddr, authToken)
	if err != nil {
		return err
	}

	err = c.VNC.Start(c.session)
	if err != nil {
		return err
	}
	c.Chat.Start(c.session)

	c.started = true
	return nil
}

func (c *clientRemoton) MachineID() string {
	if c.session == nil {
		return ""
	}
	return c.session.ID
}

func (c *clientRemoton) MachineAuth() string {
	if c.session == nil {
		return ""
	}

	return c.session.AuthToken
}

func (c *clientRemoton) Stop() {
	c.Terminate()
}

func (c *clientRemoton) Terminate() {
	c.Chat.Stop()
	c.VNC.Stop()
	if c.session != nil {
		c.session.Destroy()
	}
	c.started = false
}
