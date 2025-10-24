package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ocppSubprotocol   = "ocpp2.0"
	heartbeatInterval = 10
)

var upgrader = websocket.Upgrader{
	Subprotocols: []string{ocppSubprotocol},
	CheckOrigin:  func(r *http.Request) bool { return true },
}

func ocppHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade err:", err)
		return
	}
	defer c.Close()

	if c.Subprotocol() != ocppSubprotocol {
		log.Println("wrong subprotocol:", c.Subprotocol())
		return
	}

	log.Println("OCPP connected:", r.RemoteAddr)

	for {
		msgType, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			return
		}
		if msgType != websocket.TextMessage {
			log.Println("unexpected message type:", msgType)
			continue
		}
		log.Printf("RX: %s\n", string(msg))

		var frame []any
		if err := json.Unmarshal(msg, &frame); err != nil || len(frame) < 3 {
			log.Println("bad frame:", err)
			continue
		}

		typ, _ := frame[0].(float64)
		uid, _ := frame[1].(string)
		action, _ := frame[2].(string)
		now := time.Now().UTC().Format(time.RFC3339)

		if int(typ) == 2 {
			switch action {
			case "BootNotification":
				resp := []any{3, uid, map[string]any{
					"currentTime": now,
					"interval":    heartbeatInterval,
					"status":      "Accepted",
				}}
				if err := sendJSON(c, resp); err != nil {
					log.Println("write BootNotification err:", err)
					return
				}
				log.Printf("TX: BootNotification.Accepted (interval=%d)\n", heartbeatInterval)

			case "Heartbeat":
				resp := []any{3, uid, map[string]any{
					"currentTime": now,
				}}
				if err := sendJSON(c, resp); err != nil {
					log.Println("write Heartbeat err:", err)
					return
				}
				log.Println("TX: Heartbeat response")

				closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Heartbeat test complete")
				if err := c.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(2*time.Second)); err != nil {
					log.Println("close control err:", err)
				} else {
					log.Println("Closing connection after heartbeat test")
				}
				return

			default:
				log.Println("Unhandled action:", action)
			}
		}
	}
}

func sendJSON(c *websocket.Conn, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, b)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ocpp/", ocppHandler)
	log.Println("CSMS stub listening on :8080  (ws)  path=/ocpp/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
