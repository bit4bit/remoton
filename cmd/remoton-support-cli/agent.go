package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"io"
	"net"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/bit4bit/remoton"
)

var (
	srv        = flag.String("srv", "localhost:9934", "server address")
	tunnelAddr = flag.String("tunnel", "localhost:9959", "tunnel addres")
	service    = flag.String("service", "nx", "service")
	auth       = flag.String("auth", "", "auth session:user:pass")
	chat       = flag.Bool("chat", false, "dial to chat service")

	rclient = &remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}
)

func main() {
	var sessionID string
	var sessionAuth string
	flag.Parse()

	if *auth != "" {
		parse := strings.Split(*auth, ":")
		sessionID = parse[0]
		sessionAuth = parse[1]
	}

	session := &remoton.SessionClient{Client: rclient,
		ID: sessionID, AuthToken: sessionAuth,
		APIURL: "https://" + *srv}

	if *chat {
		wsconnChat, err := session.Dial("chat")
		if err != nil {
			log.Fatal(err)
		}
		defer wsconnChat.Close()
		go chatStd(wsconnChat)
	}

	log.Println("connected vnc")
	listen, err := net.Listen("tcp", *tunnelAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		log.Println("waiting client vnc")
		conn, err := listen.Accept()
		if err != nil {
			log.Error(err)
			break
		}
		wsconn, err := session.Dial(*service)
		if err != nil {
			log.Error(err)
			break
		}

		go handleConn(conn, wsconn)
	}

}

func handleConn(local, remoto net.Conn) {
	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(local, remoto)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(remoto, local)
		errc <- err
	}()
	log.Info("processing..")

	log.Error(<-errc)

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
