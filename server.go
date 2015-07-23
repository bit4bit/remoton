package remoton

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
)

const BASE_API_URL = "/remoton"

type requestTunnel struct {
	SessionID string
	Service   string
}

var (
	chListenTunnel     = make(chan requestTunnel)
	chAcceptTunnel     = make(chan net.Conn)
	chDialTunnel       = make(chan requestTunnel)
	chDialAcceptTunnel = make(chan net.Conn)
)

//Server export API for handle connections
//this follow http.Handler can be embedded in any
//web app
type Server struct {
	*httprouter.Router

	//sessions handle sessions any session can have only 2 connections
	//one for the producer and one for the consumer
	sessions      *sessionManager
	authGenerator func() (string, string)
}

//New create a new Server it't interface http.Handler
//can handle with http.ListenAndServer
func NewServer(authToken string, authGenerator func() (string, string)) *Server {
	r := &Server{httprouter.New(), NewSessionManager(), authGenerator}
	r.POST(BASE_API_URL+"/session", hAuth(authToken, r.hNewSession))

	//DELETE active session this not close active connections
	r.DELETE(BASE_API_URL+"/session/:id", r.hDestroySession)
	r.GET(BASE_API_URL+"/session/:id/conn/:service/dial", r.hSessionDial)
	r.GET(BASE_API_URL+"/session/:id/conn/:service/listen", r.hSessionListen)

	return r
}

//hNewSession create a session and return ID:USERNAME:PASSWORD
func (c *Server) hNewSession(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	id, authSession := c.authGenerator()

	c.sessions.Add(id, newSession(id, authSession))

	resp := struct {
		ID        string
		AuthToken string
	}{
		ID:        id,
		AuthToken: authSession,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	if _, err := w.Write(data); err != nil {
		log.Error(err)
	}
}

//hDestroySession destroy a session
//need header *X-Auth-Username* and *X-Auth-Password*
func (c *Server) hDestroySession(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	if session := c.sessions.Get("id"); session != nil {
		if !authenticateSession(session, r) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		c.sessions.Del(params.ByName("id"))
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (c *Server) hSessionDial(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var tunnel net.Conn

	session := c.sessions.Get(params.ByName("id"))
	if session == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !authenticateSession(session, r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	kservice := params.ByName("service")
	tunnel = session.DialService(kservice)

	log.Infof("Dial service %s for %s", kservice, params.ByName("id"))
	wsCopy(tunnel).ServeHTTP(w, r)

}

func (c *Server) hSessionListen(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var tunnel net.Conn

	session := c.sessions.Get(params.ByName("id"))
	if session == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !authenticateSession(session, r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	kservice := params.ByName("service")
	log.Infof("Listen service %s for %s", kservice, params.ByName("id"))

	tunnel = <-session.ListenService(kservice)

	wsCopy(tunnel).ServeHTTP(w, r)
}

func authenticateSession(session *srvSession, r *http.Request) bool {
	return session.ValidateAuthToken(r.Header.Get("X-Auth-Session"))
}

func wsCopy(src io.ReadWriteCloser) websocket.Server {

	handshake := func(conf *websocket.Config, r *http.Request) error {
		conf.Protocol = []string{"binary"}
		return nil
	}

	handler := websocket.Handler(func(ws *websocket.Conn) {
		ws.PayloadType = websocket.BinaryFrame
		err := <-connTunnel(src, ws)
		log.Error("wsCopy:", err)
		ws.Close()
		src.Close()
	})

	ws := websocket.Server{
		Handshake: handshake,
		Handler:   handler,
	}

	return ws
}

func connTunnel(dst io.ReadWriteCloser, src io.ReadWriteCloser) chan error {
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

func hAuth(authToken string, handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Header.Get("X-Auth-Token") != authToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler(w, r, params)
	}
}
