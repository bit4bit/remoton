package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/bit4bit/remoton"
)

type Session struct {
	ID        string
	AuthToken string
}

func (c Session) String() string {
	return fmt.Sprintf("Session[%s]=%s \n\t%s:%s",
		c.ID, c.AuthToken,
		c.ID, c.AuthToken)
}

var (
	srv        = flag.String("srv", "localhost:9934", "server address")
	service    = flag.String("service", "chat", "service chat")
	tunnelAddr = flag.String("tunnel", "localhost:5900", "tunnel address")
	authToken  = flag.String("auth", "", "auth token")
	srvPrefix  = flag.String("srv-prefix", "/remoton", "base app default remoton")
	chat       = flag.Bool("chat", false, "dial to chat service")
)

func main() {
	flag.Parse()

	if *authToken == "" {
		log.Error("need auth token please use -auth")
		return
	}

	rclient := remoton.Client{Prefix: *srvPrefix, TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}
	session, err := rclient.NewSession("https://"+*srv, *authToken)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Session -> %s:%s", session.ID, session.AuthToken)
	defer session.Destroy()

	if *chat {
		log.Println("Enable terminal chat")
		lChat := session.Listen("chat")

		go func(listener net.Listener) {
			for {
				wsconn, err := listener.Accept()
				if err != nil {
					log.Error("chat:", err)
					break
				}
				go chatStd(wsconn)
			}
		}(lChat)

	}

	listener := session.Listen(*service)
	if err != nil {
		log.Fatal(err)
	}

	for {
		wsconn, err := listener.Accept()
		if err != nil {
			log.Error(err)
			break
		}

		go func(wsconn net.Conn) {
			conn, err := net.Dial("tcp", *tunnelAddr)
			if err != nil {
				listener.Close()
				log.Error(err)
				return
			}

			errc := make(chan error, 2)
			cp := func(dst net.Conn, src net.Conn) {
				_, err := io.Copy(dst, src)
				errc <- err
			}

			go cp(conn, wsconn)
			go cp(wsconn, conn)

			log.Error(<-errc)
			conn.Close()
			wsconn.Close()
		}(wsconn)
	}
}

func chatStd(conn net.Conn) {
	input := bufio.NewReader(os.Stdin)
	output := bufio.NewWriter(conn)

	go func() {
		for {
			buf := make([]byte, 32*512)
			_, err := conn.Read(buf)
			if err != nil {
				break
			}
			os.Stderr.WriteString("remote chat <- " + string(buf) + "\n")
		}
	}()
	for {
		msg, err := input.ReadString('\n')
		if err != nil {
			log.Error(err)
			break
		}
		output.WriteString(msg)
		output.Flush()
	}

}
