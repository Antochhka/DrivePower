package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"ocpp2.0.0"},
	CheckOrigin:  func(r *http.Request) bool { return true }, // для локальной отладки
}

func ocppHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {.й
		log.Println("upgrade err:", err)
		return
	}
	defer c.Close()

	if c.Subprotocol() != "ocpp2.0.1" {
		log.Println("wrong subprotocol:", c.Subprotocol())
		return
	}

	log.Println("OCPP connected:", r.RemoteAddr)

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			return
		}
		log.Printf("RX: %s\n", string(msg))

		var frame []any
		if err := json.Unmarshal(msg, &frame); err != nil || len(frame) < 3 {
			log.Println("bad frame:", err)
			continue
		}

		typ, _ := frame[0].(float64)   // 2=Call, 3=Result, 4=Error
		uid, _ := frame[1].(string)    // UniqueId
		action, _ := frame[2].(string) // "BootNotification", "Heartbeat", ...
		now := time.Now().UTC().Format(time.RFC3339)

		if int(typ) == 2 { // Call
			switch action {
			case "BootNotification":
				resp := []any{3, uid, map[string]any{
					"currentTime": now,
					"interval":    300, // 5 минут
					"status":      "Accepted",
				}}
				b, _ := json.Marshal(resp)
				_ = c.WriteMessage(websocket.TextMessage, b)
				log.Println("TX: BootNotification.Accepted")

			case "Heartbeat":
				resp := []any{3, uid, map[string]any{
					"currentTime": now,
				}}
				b, _ := json.Marshal(resp)
				_ = c.WriteMessage(websocket.TextMessage, b)
				log.Println("TX: Heartbeat")

			default:
				// на прочие вызовы пока не отвечаем
				log.Println("Unhandled action:", action)
			}
		}
	}
}

func main() {
	http.HandleFunc("/ocpp/", ocppHandler)
	log.Println("CSMS stub listening on :8080  (ws)  path=/ocpp/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
