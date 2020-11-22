package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/template"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

func joinURL(uuid string, ws bool) string {
	var (
		protocol = "http"
		route    = "join"
	)
	if ws {
		protocol = "ws"
		route = "ws"
	}

	return fmt.Sprintf("%s://localhost:8080/%s/%s", protocol, route, uuid)
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

func create(w http.ResponseWriter, r *http.Request) {
	serverChan := make(chan chan message, 4)

	uuid := uuid.New().String()
	sessionsMutex.Lock()
	sessions[uuid] = serverChan
	sessionsMutex.Unlock()

	sessionLog(uuid, "Creating")

	conn, _, _, err := ws.UpgradeHTTP(r, w)
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
		sendMessage(w, encoder, message{messageTypeURL, joinURL(uuid, false)})

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
				sendMessage(w, encoder, message{messageTypeResend, ""})
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
				sessionsMutex.Lock()
				delete(sessions, uuid)
				sessionsMutex.Unlock()

				close(serverChan)

				// Close listeners
				clientsMutex.Lock()
				for _, c := range clients {
					close(c)
				}
				clientsMutex.Unlock()

				sessionLog(uuid, "Closed")

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

func joinWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
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

	conn, _, _, err := ws.UpgradeHTTP(r, w)
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
func join(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
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

	t.Execute(w, joinURL(uuid, true))
}

func main() {
	sessions = make(map[string]chan chan message)

	router := mux.NewRouter()
	router.HandleFunc("/create", create)
	router.HandleFunc("/join/{uuid}", join)
	router.HandleFunc("/ws/{uuid}", joinWS)

	// Setup file server
	fs := http.FileServer(http.Dir("public"))
	router.PathPrefix("/public").Handler(http.StripPrefix("/public/", fs))

	http.Handle("/", router)
	http.ListenAndServe(":8080", nil)
}
