package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type message struct {
	// the json tag means this will serialize as a lowercased field
	Message string `json:"message"`
}

var sessions map[string]chan chan string

func sendMessage(w *wsutil.Writer, encoder *json.Encoder, msg string) bool {
	if err := encoder.Encode(&message{msg}); err != nil {
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

func create(w http.ResponseWriter, r *http.Request) {
	serverChan := make(chan chan string, 4)

	uuid := uuid.New().String()
	sessions[uuid] = serverChan

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

			clients []chan string
		)

		// Send UUID
		sendMessage(w, encoder, fmt.Sprintf("ws://localhost:8080/join/%s", uuid))

		go func() {
			for {
				// Listen for new clients
				client := <-serverChan
				clients = append(clients, client)
				sessionLog(uuid, "Detected new client")
			}
		}()

		for {
			// Await some data from main code server
			hdr, err := r.NextFrame()
			if err != nil {
				// no data, skip
				continue
			}

			if hdr.OpCode == ws.OpClose {
				// Close listening clients somehow?
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

			fmt.Println("Received:", msg.Message)
			for _, c := range clients {
				c <- msg.Message
			}
		}
	}()
}

func join(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	serverChan, ok := sessions[uuid]
	if !ok {
		sessionLog(uuid, "No session found")
		w.Write([]byte(fmt.Sprintf("[%s] No session found", uuid)))
		return
	}

	sessionLog(uuid, "Joining")

	client := make(chan string, 1)
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

		// Wait for new data to send indefinitely
		for {
			text := <-client
			sendMessage(w, encoder, text)
		}
	}()
}

func main() {
	sessions = make(map[string]chan chan string)
	router := mux.NewRouter()

	router.HandleFunc("/create", create)
	router.HandleFunc("/join/{uuid}", join)

	http.Handle("/", router)
	http.ListenAndServe(":8080", nil)
}
