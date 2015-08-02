package main

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/bit4bit/remoton"
	"github.com/bit4bit/remoton/common"
	"github.com/bit4bit/remoton/xpra"
)

type chatRemoton struct {
	onRecv func(msg string)
	chRecv chan string
	conn   net.Conn
}

func (c *chatRemoton) init() {
	if c.chRecv == nil {
		c.chRecv = make(chan string)
	}
}

func (c *chatRemoton) Start(session *remoton.SessionClient) error {
	chatConn, err := session.Dial("chat")
	if err != nil {
		return err
	}
	c.conn = chatConn
	c.init()
	go c.handle()
	return nil
}

func (c *chatRemoton) handle() {

	for {
		buf := make([]byte, 32*512)
		log.Println("waiting")
		rlen, err := c.conn.Read(buf)
		if err != nil {
			log.Error(err)
			break
		}

		if c.onRecv != nil {
			c.onRecv(strings.TrimSpace(string(buf[0:rlen])))
		}
	}
}

func (c *chatRemoton) Send(msg string) {
	if c.conn != nil {
		c.conn.Write([]byte(msg))
	}
}

func (c *chatRemoton) OnRecv(f func(msg string)) {
	c.onRecv = f
}

func (c *chatRemoton) Terminate() {
	if c.conn != nil {
		c.conn.Close()
	}
}

type tunnelRemoton struct {
	listener net.Listener
}

func (c *tunnelRemoton) Start(session *remoton.SessionClient) error {

	rpconn, err := session.Dial("rpc")
	if err != nil {
		return err
	}
	defer rpconn.Close()

	rpcclient := rpc.NewClient(rpconn)
	defer rpcclient.Close()

	var capsClient common.Capabilities
	err = rpcclient.Call("RemotonClient.GetCapabilities", struct{}{}, &capsClient)
	if err != nil {
		return err
	}
	if capsClient.XpraVersion != xpra.Version() {
		return fmt.Errorf("mismatch xpra version was %s expected %s",
			capsClient.XpraVersion, xpra.Version())
	}

	var clientExternalIP net.IP
	err = rpcclient.Call("RemotonClient.GetExternalIP", struct{}{}, &clientExternalIP)
	if err == nil {
		log.Println(clientExternalIP)
		return c.srvDirect(session, clientExternalIP)
	}

	return c.srvTunnel(session)
}

func (c *tunnelRemoton) srvDirect(session *remoton.SessionClient, externalIP net.IP) error {
	log.Println("direct connection")
	//TODO xpra connection by tcp it's insecure
	err := xpra.Attach(net.JoinHostPort(externalIP.String(), "9932"))
	if err != nil {
		return err
	}
	return nil
}

func (c *tunnelRemoton) srvTunnel(session *remoton.SessionClient) error {
	port := c.findFreePort()
	addrSrv := "localhost:" + port
	log.Println("listen at " + addrSrv)
	listener, err := net.Listen("tcp", addrSrv)
	if err != nil {
		return err
	}
	c.listener = listener

	go func(listener net.Listener) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				listener.Close()
				log.Error(err)
				break
			}
			remote, err := session.DialTCP("nx")
			if err != nil {
				log.Error(err)
				listener.Close()

				break
			}
			log.Println("new connection")
			go c.handle(conn, remote)
		}
	}(listener)

	err = xpra.Attach(addrSrv)
	if err != nil {
		listener.Close()
		return err
	}
	return nil
}

func (c *tunnelRemoton) handle(local, remoto net.Conn) {

	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(local, remoto)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(remoto, local)
		errc <- err
	}()

	log.Error(<-errc)
	local.Close()
	remoto.Close()
}

func (c *tunnelRemoton) findFreePort() string {
	startPort := 55123

	for ; startPort < 65534; startPort++ {
		conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(startPort))
		if err != nil {
			return strconv.Itoa(startPort)
		}
		conn.Close()
	}
	panic("cant find free port")
}

func (c *tunnelRemoton) Terminate() {
	if c.listener != nil {
		c.listener.Close()
	}
	xpra.Terminate()
}
