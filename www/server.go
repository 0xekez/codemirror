package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Regex compilations
var (
	createRe = regexp.MustCompile(`/create.*$`)
)

const (
	messageTypeData      = "DATA"
	messageTypeURL       = "URL"
	messageTypeResend    = "RESEND"
	messageTypeSelection = "SELECTION"
)

type session struct {
	uuid           string
	alive          bool
	editorChan     chan chan message
	listeners      []chan message
	listenersMutex sync.Mutex
}

type message struct {
	// the json tag means this will serialize as a lowercased field
	Type    string `json:"type"`
	Content string `json:"content"`
}

var sessions map[string]*session
var sessionsMutex sync.Mutex

func sendMessage(w *wsutil.Writer, encoder *json.Encoder, msg message) bool {
	if err := encoder.Encode(&msg); err != nil {
		fmt.Println("Encode err:", err)
		return false
	}

	if err := w.Flush(); err != nil {
		fmt.Println("Flush err:", err)
		return false
	}

	return true
}

func (s *session) log(args ...string) {
	fmt.Println(fmt.Sprintf("[%s]", s.uuid), strings.Join(args, " "))
}

func indexOf(chans []chan message, ch chan message) int {
	for idx, elem := range chans {
		if ch == elem {
			return idx
		}
	}
	return -1
}

func remove(chans []chan message, i int) []chan message {
	chans[len(chans)-1], chans[i] = chans[i], chans[len(chans)-1]
	return chans[:len(chans)-1]
}

/**
 * Deletes the session from the master map, closes the server channel, and closes the client channels.
 */
func (s *session) close() {
	// Lock clients first so nothing can try to send through
	// a channel while the session is being closed.
	s.listenersMutex.Lock()

	sessionsMutex.Lock()
	delete(sessions, s.uuid)
	sessionsMutex.Unlock()

	close(s.editorChan)

	s.alive = false

	// Close listeners
	for _, c := range s.listeners {
		close(c)
	}
	s.listenersMutex.Unlock()

	s.log("Closed")
}

func create(writer http.ResponseWriter, req *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(req, writer)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	session := session{
		alive:      true,
		editorChan: make(chan chan message, 3),
	}

	sessionsMutex.Lock()
	/* Create UUID and ensure it is not in use. */
	for {
		session.uuid = uuid.New().String()
		if _, ok := sessions[session.uuid]; !ok {
			break
		}
	}

	sessions[session.uuid] = &session
	sessionsMutex.Unlock()

	session.log("Creating")

	var (
		r       = wsutil.NewReader(conn, ws.StateServerSide)
		w       = wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
		decoder = json.NewDecoder(r)
		encoder = json.NewEncoder(w)
	)

	// Listen for new or closed listeners
	go func() {
		defer session.log("Listeners func ended")

		for {
			listener, open := <-session.editorChan
			// Session closed
			if !open || !session.alive {
				return
			}

			session.listenersMutex.Lock()
			// If channel already exists in list,
			// broken pipe detected so we should CLOSE.
			idx := indexOf(session.listeners, listener)
			if idx > -1 {
				session.listeners = remove(session.listeners, idx)
				close(listener)
				session.log("Listener left")

				session.listenersMutex.Unlock()
				continue
			}

			session.listeners = append(session.listeners, listener)
			session.listenersMutex.Unlock()

			session.log("Setup new listener")

			// Ask editor to re-broadcast latest code
			if !sendMessage(w, encoder, message{messageTypeResend, ""}) {
				// if could not send successfully, close server
				session.close()
				return
			}
		}
	}()

	go func() {
		defer session.log("Forwarder func ended")
		defer conn.Close()

		// Send UUID join URL
		url := fmt.Sprintf("http://%s%s", req.Host, createRe.ReplaceAllString(req.URL.RequestURI(), fmt.Sprintf("/join/%s", session.uuid)))
		sendMessage(w, encoder, message{messageTypeURL, url})

		for {
			// Await some data from editor (host)
			hdr, err := r.NextFrame()
			if err != nil {
				if !session.alive {
					return
				}
				// no data, skip
				continue
			}

			if hdr.OpCode == ws.OpClose {
				session.close()
				return
			}

			var msg message
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF || !session.alive {
					return
				}
				fmt.Println("Decode err:", err)
				continue
			}

			session.log("Received:", msg.Type)
			// Forward message to listeners
			session.listenersMutex.Lock()
			for _, l := range session.listeners {
				l <- msg
			}
			session.listenersMutex.Unlock()
		}
	}()
}

func joinWS(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	sessionsMutex.Lock()
	session, ok := sessions[uuid]
	sessionsMutex.Unlock()
	if !ok || !session.alive {
		fmt.Println("No session found for", uuid)
		w.Write([]byte(fmt.Sprintf("No session found for %s", uuid)))
		return
	}

	session.log("Joining")

	listener := make(chan message, 1)
	session.editorChan <- listener

	conn, _, _, err := ws.UpgradeHTTP(req, w)
	if err != nil {
		fmt.Println("Upgrade join err:", err)
		return
	}

	go func() {
		defer session.log("Joined listener func ended")
		defer conn.Close()

		var (
			state = ws.StateServerSide
			// r     = wsutil.NewReader(conn, state)
			w = wsutil.NewWriter(conn, state, ws.OpText)

			// decoder = json.NewDecoder(r)
			encoder = json.NewEncoder(w)
		)

		// Wait for new data to send
		for {
			msg, open := <-listener
			if !open || !session.alive {
				// Session closed if channel is closed
				return
			}

			// if could not send successfully, close listener
			if !sendMessage(w, encoder, msg) {
				// A second send to this channel triggers removal from the list and closure of the channel
				session.editorChan <- listener
				return
			}
		}
	}()
}

// Web interface
func join(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	sessionsMutex.Lock()
	s, ok := sessions[uuid]
	sessionsMutex.Unlock()
	if !ok || !s.alive {
		fmt.Println("No session found for", uuid)
		w.Write([]byte(fmt.Sprintf("No session found for %s", uuid)))
		return
	}

	t, err := template.ParseFiles("./templates/join.html")
	if err != nil {
		fmt.Println("Template parse err:", err)
		w.Write([]byte("An unexpected error occurred"))
	}

	url := fmt.Sprintf("wss://%s%s", req.Host, strings.ReplaceAll(req.URL.RequestURI(), fmt.Sprintf("/join/%s", uuid), fmt.Sprintf("/ws/%s", uuid)))
	t.Execute(w, url)
}

func main() {
	fmt.Println("Starting...")

	sessions = make(map[string]*session)

	router := mux.NewRouter()
	router.HandleFunc("/create", create)
	router.HandleFunc("/join/{uuid}", join)
	router.HandleFunc("/ws/{uuid}", joinWS)

	// Setup file server
	fs := http.FileServer(http.Dir("public"))
	router.PathPrefix("/public").Handler(http.StripPrefix("/public/", fs))

	http.Handle("/", router)

	fmt.Println("Started. Listening...")
	http.ListenAndServe(":7654", nil)
	fmt.Println("Stopping...")
}
