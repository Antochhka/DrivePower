package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

	stationID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/ocpp/"), "/")
	if stationID == "" {
		stationID = "unknown"
	}

	log.Printf("OCPP connected: %s (station=%s path=%s)\n", r.RemoteAddr, stationID, r.URL.Path)

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
		log.Printf("RX <- %s: %s\n", stationID, string(msg))

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
				vendor, model, reason := parseBootNotification(frame)
				log.Printf("BootNotification received from %s: vendor=%s model=%s reason=%s\n", stationID, vendor, model, reason)

				resp := []any{3, uid, map[string]any{
					"currentTime": now,
					"interval":    heartbeatInterval,
					"status":      "Accepted",
				}}
				if err := sendJSON(c, resp); err != nil {
					log.Println("write BootNotification err:", err)
					return
				}
				log.Printf("TX -> %s: BootNotification.Accepted (interval=%d)\n", stationID, heartbeatInterval)

			case "Heartbeat":
				log.Printf("Heartbeat received from %s\n", stationID)
				resp := []any{3, uid, map[string]any{
					"currentTime": now,
				}}
				if err := sendJSON(c, resp); err != nil {
					log.Println("write Heartbeat err:", err)
					return
				}
				log.Printf("TX -> %s: Heartbeat response\n", stationID)

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

func parseBootNotification(frame []any) (vendor, model, reason string) {
	if len(frame) < 4 {
		return "", "", ""
	}

	payload, _ := frame[3].(map[string]any)
	if payload != nil {
		if v, ok := payload["reason"].(string); ok {
			reason = v
		}
		if chargingStation, ok := payload["chargingStation"].(map[string]any); ok {
			if v, ok := chargingStation["vendorName"].(string); ok {
				vendor = v
			}
			if m, ok := chargingStation["model"].(string); ok {
				model = m
			}
			if serial, ok := chargingStation["serialNumber"].(string); ok && serial != "" {
				model = fmt.Sprintf("%s (S/N %s)", model, serial)
			}
		}
	}

	return vendor, model, reason
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ocpp/", ocppHandler)
	log.Println("CSMS stub listening on :8080  (ws)  path=/ocpp/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
