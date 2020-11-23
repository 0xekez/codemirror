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

type message struct {
	// the json tag means this will serialize as a lowercased field
	Type    string `json:"type"`
	Content string `json:"content"`
}

var sessions map[string]chan chan message
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

func sessionLog(uuid string, args ...string) {
	fmt.Println(fmt.Sprintf("[%s]", uuid), strings.Join(args, " "))
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
func closeSession(uuid string, serverChan chan chan message, clients []chan message, clientsMutex *sync.Mutex) {
	// Lock clients first so nothing can try to send through
	// a channel while the session is being closed.
	clientsMutex.Lock()

	sessionsMutex.Lock()
	delete(sessions, uuid)
	sessionsMutex.Unlock()

	close(serverChan)

	// Close listeners
	for _, c := range clients {
		close(c)
	}
	clientsMutex.Unlock()

	sessionLog(uuid, "Closed")
}

func create(w http.ResponseWriter, req *http.Request) {
	serverChan := make(chan chan message, 4)

	uuid := uuid.New().String()
	sessionsMutex.Lock()
	sessions[uuid] = serverChan
	sessionsMutex.Unlock()

	sessionLog(uuid, "Creating")

	conn, _, _, err := ws.UpgradeHTTP(req, w)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}
	go func() {
		defer conn.Close()

		var (
			state = ws.StateServerSide
			r     = wsutil.NewReader(conn, state)
			w     = wsutil.NewWriter(conn, state, ws.OpText)

			decoder = json.NewDecoder(r)
			encoder = json.NewEncoder(w)

			clients      []chan message
			clientsMutex sync.Mutex
		)

		// Send UUID join URL
		url := fmt.Sprintf("http://%s%s", req.Host, createRe.ReplaceAllString(req.URL.RequestURI(), fmt.Sprintf("/join/%s", uuid)))
		sendMessage(w, encoder, message{messageTypeURL, url})

		// Listen for new clients or for closed clients
		go func() {
			for {
				client, open := <-serverChan
				if !open {
					// Session closed if channel is closed
					return
				}

				clientsMutex.Lock()
				// If channel already exists in list,
				// broken pipe detected so we should CLOSE.
				idx := indexOf(clients, client)
				if idx > -1 {
					clients = remove(clients, idx)
					close(client)
					sessionLog(uuid, "Client left")

					clientsMutex.Unlock()
					continue
				}

				clients = append(clients, client)
				clientsMutex.Unlock()

				sessionLog(uuid, "Detected new client")

				// Ask editor to re-broadcast latest code
				if !sendMessage(w, encoder, message{messageTypeResend, ""}) {
					// if could not send successfully, close server
					closeSession(uuid, serverChan, clients, &clientsMutex)
					return
				}
			}
		}()

		for {
			// Await some data from editor (host)
			hdr, err := r.NextFrame()
			if err != nil {
				// no data, skip
				continue
			}

			if hdr.OpCode == ws.OpClose {
				closeSession(uuid, serverChan, clients, &clientsMutex)
				return
			}

			var msg message
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					return
				}
				fmt.Println("Decode err:", err)
				continue
			}

			sessionLog(uuid, "Received:", msg.Type)
			// Forward message to listeners
			clientsMutex.Lock()
			for _, c := range clients {
				c <- msg
			}
			clientsMutex.Unlock()
		}
	}()
}

func joinWS(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	sessionsMutex.Lock()
	serverChan, ok := sessions[uuid]
	sessionsMutex.Unlock()
	if !ok {
		sessionLog(uuid, "No session found")
		w.Write([]byte(fmt.Sprintf("[%s] No session found", uuid)))
		return
	}

	sessionLog(uuid, "Joining")

	client := make(chan message, 1)
	serverChan <- client

	conn, _, _, err := ws.UpgradeHTTP(req, w)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	go func() {
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
			msg, open := <-client
			if !open {
				// Session closed if channel is closed
				return
			}

			// if could not send successfully, close client
			if !sendMessage(w, encoder, msg) {
				// A second send to this channel triggers removal from the list and closure of the channel
				serverChan <- client

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
	_, ok := sessions[uuid]
	sessionsMutex.Unlock()
	if !ok {
		sessionLog(uuid, "No session found")
		w.Write([]byte(fmt.Sprintf("No session found for ID %s", uuid)))
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

	sessions = make(map[string]chan chan message)

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
