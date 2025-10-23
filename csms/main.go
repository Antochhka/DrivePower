package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"charging_station/internal/ws"
)

const (
	ocppSubprotocol   = "ocpp2.0"
	heartbeatInterval = 10
)

var upgrader = ws.Upgrader{
	Subprotocols: []string{ocppSubprotocol},
	CheckOrigin:  func(r *http.Request) bool { return true }, // для локальной отладки
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
		if msgType != ws.TextMessage {
			log.Println("unexpected message type:", msgType)
			continue
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

				closeDeadline := time.Now().Add(2 * time.Second)
				closeMsg := ws.FormatCloseMessage(1000, "Heartbeat test complete")
				if err := c.WriteControl(ws.CloseMessage, closeMsg, closeDeadline); err != nil {
					log.Println("close control err:", err)
				} else {
					log.Println("Closing connection after heartbeat test")
				}
				return

			default:
				// на прочие вызовы пока не отвечаем
				log.Println("Unhandled action:", action)
			}
		}
	}
}

func sendJSON(c *ws.Conn, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.WriteMessage(ws.TextMessage, b)
}

func main() {
	http.HandleFunc("/ocpp/", ocppHandler)
	log.Println("CSMS stub listening on :8080  (ws)  path=/ocpp/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
