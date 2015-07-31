package remoton

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
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
	r.GET(BASE_API_URL+"/session/:id/conn/:service/dial/:tunnel", r.hSessionDial)
	r.GET(BASE_API_URL+"/session/:id/conn/:service/listen/:tunnel", r.hSessionListen)

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
	if trans, ok := tunnelTypes[params.ByName("tunnel")]; ok {
		log.Infof("Dial Translator %s activated for service %s.",
			params.ByName("tunnel"),
			params.ByName("service"))
		tunnel := session.DialService(kservice)
		defer tunnel.Close()
		log.Infof("Executing translator")
		trans(tunnel).ServeHTTP(w, r)
		log.Info("Ending Translator")

		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func (c *Server) hSessionListen(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	session := c.sessions.Get(params.ByName("id"))
	if session == nil {
		log.Info("Invalid session")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !authenticateSession(session, r) {
		log.Info("Invalid authentication on listen")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	kservice := params.ByName("service")
	log.Infof("Listen service %s for %s", kservice, params.ByName("id"))

	if trans, ok := tunnelTypes[params.ByName("tunnel")]; ok {
		log.Infof("Listener Translator %s activated.", params.ByName("tunnel"))
		tunnel := <-session.ListenService(kservice)
		defer tunnel.Close()
		log.Infof("Executing translator")
		trans(tunnel).ServeHTTP(w, r)
		log.Info("Ending Translator")
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func authenticateSession(session *srvSession, r *http.Request) bool {
	return session.ValidateAuthToken(r.Header.Get("X-Auth-Session"))
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
