package remoton

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

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

var tunnelTypes map[string]func(net.Conn) http.Handler

//RegisterTranslatorType for handling connections
func RegisterTunnelType(typ string, f func(net.Conn) http.Handler) {
	if tunnelTypes == nil {
		tunnelTypes = make(map[string]func(net.Conn) http.Handler)
	}
	tunnelTypes[typ] = f
}

//Server export API for handle connections
//this follow http.Handler can be embedded in any
//web app
type Server struct {
	*httprouter.Router

	//sessions handle sessions any session can have only 2 connections
	//one for the producer and one for the consumer
	sessions    *sessionManager
	idGenerator func() string
}

//New create a new Server it't interface http.Handler
//can handle with http.ListenAndServer
func NewServer(authToken string, idGenerator func() string) *Server {
	r := &Server{httprouter.New(), NewSessionManager(), idGenerator}
	r.RedirectFixedPath = false

	r.POST("/session", hAuth(authToken, r.hNewSession))
	//DELETE active session this not close active connections
	r.DELETE("/session/:id", r.hDestroySession)
	r.GET("/session/:id/conn/:service/dial/:tunnel", r.hSessionDial)
	r.GET("/session/:id/conn/:service/listen/:tunnel", r.hSessionListen)

	return r
}

//hNewSession create a session and return ID:USERNAME:PASSWORD
func (c *Server) hNewSession(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id := c.idGenerator()

	c.sessions.Add(id, newSession(id))

	resp := struct {
		ID string
	}{
		ID: id,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//hDestroySession destroy a session
//need header *X-Auth-Username* and *X-Auth-Password*
func (c *Server) hDestroySession(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if session := c.sessions.Get("id"); session != nil {
		c.sessions.Del(params.ByName("id"))
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (c *Server) hSessionDial(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	session := c.sessions.Get(params.ByName("id"))
	if session == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	kservice := params.ByName("service")
	if trans, ok := tunnelTypes[params.ByName("tunnel")]; ok {
		tunnel := session.DialService(kservice)
		defer tunnel.Close()
		trans(tunnel).ServeHTTP(w, r)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func (c *Server) hSessionListen(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	session := c.sessions.Get(params.ByName("id"))
	if session == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	kservice := params.ByName("service")

	if trans, ok := tunnelTypes[params.ByName("tunnel")]; ok {
		tunnel := <-session.ListenService(kservice)
		defer tunnel.Close()
		trans(tunnel).ServeHTTP(w, r)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
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
