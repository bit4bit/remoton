package remoton

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

const (
	timeoutDefaultListen = time.Minute * 20
	timeoutDefaultDial   = time.Minute * 3
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

// RegisterTunnelType for handling type of connections
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

//NewServer create a new http.Listener, *authFunc* for custom authentication and
//idGenerator for identify connections
func NewServer(authFunc func(authToken string, r *http.Request) bool, idGenerator func() string) *Server {
	r := &Server{httprouter.New(), NewSessionManager(), idGenerator}
	r.RedirectFixedPath = false

	r.POST("/session", hAuth(authFunc, r.hNewSession))
	r.DELETE("/session/:id", hAuth(authFunc, r.hDestroySession))
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
		listen, tunnel := net.Pipe()
		service := session.Service(kservice)
		select {
		case service <- listen:
			defer tunnel.Close()
			trans(tunnel).ServeHTTP(w, r)
			return
		case <-time.After(timeoutDefaultDial):
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
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
		chtunnel := session.Service(kservice)
		select {
		case tunnel := <-chtunnel:
			defer tunnel.Close()
			trans(tunnel).ServeHTTP(w, r)
			return
		case <-time.After(timeoutDefaultListen):
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}

	}

	w.WriteHeader(http.StatusInternalServerError)
}

func hAuth(authTokenFunc func(authToken string, r *http.Request) bool, handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if !authTokenFunc(r.Header.Get("X-Auth-Token"), r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler(w, r, params)
	}
}
