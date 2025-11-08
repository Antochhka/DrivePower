package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"charging_station/internal/registry"
	"charging_station/internal/storage"
)

const (
	ocppSubprotocol   = "ocpp2.0"
	heartbeatInterval = 10
)

var upgrader = websocket.Upgrader{
	Subprotocols: []string{ocppSubprotocol},
	CheckOrigin:  func(r *http.Request) bool { return true },
}

func newOCPPHandler(repo storage.StationRepository, connectors *registry.Registry, events chan<- registry.StatusEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		ctx := r.Context()

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
			now := time.Now().UTC()
			nowString := now.Format(time.RFC3339)

			if int(typ) != 2 {
				continue
			}

			switch action {
			case "BootNotification":
				vendor, model, reason := parseBootNotification(frame)
				log.Printf("BootNotification received from %s: vendor=%s model=%s reason=%s\n", stationID, vendor, model, reason)

				if err := repo.UpsertBoot(ctx, storage.StationBootInfo{
					StationID: stationID,
					Vendor:    vendor,
					Model:     model,
					Reason:    reason,
					Time:      now,
				}); err != nil {
					log.Printf("station upsert err for %s: %v\n", stationID, err)
				} else {
					log.Printf("station %s stored/updated in database\n", stationID)
				}

				resp := []any{3, uid, map[string]any{
					"currentTime": nowString,
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
					"currentTime": nowString,
				}}
				if err := sendJSON(c, resp); err != nil {
					log.Println("write Heartbeat err:", err)
					return
				}
				log.Printf("TX -> %s: Heartbeat response\n", stationID)

				if err := repo.UpdateLastSeen(ctx, stationID, now); err != nil {
					log.Printf("update last_seen_at err for %s: %v\n", stationID, err)
				} else {
					log.Printf("station %s last_seen_at updated to %s\n", stationID, nowString)
				}

				log.Printf("Heartbeat loop continues for %s\n", stationID)
				continue

			case "StatusNotification":
				update, err := parseStatusNotification(frame)
				if err != nil {
					log.Printf("StatusNotification parse err from %s: %v\n", stationID, err)
					errPayload := []any{4, uid, "FormationViolation", err.Error(), map[string]any{}}
					if sendErr := sendJSON(c, errPayload); sendErr != nil {
						log.Println("write StatusNotification error response err:", sendErr)
						return
					}
					continue
				}

				update.Timestamp = update.Timestamp.UTC()
				event, err := connectors.Update(stationID, update, now)
				if err != nil {
					log.Printf("StatusNotification update err for %s: %v\n", stationID, err)
					errPayload := []any{4, uid, "InternalError", err.Error(), map[string]any{}}
					if sendErr := sendJSON(c, errPayload); sendErr != nil {
						log.Println("write StatusNotification error response err:", sendErr)
						return
					}
					continue
				}

				persistErr := repo.UpsertConnectorStatus(ctx, storage.ConnectorStatusRecord{
					StationID:         stationID,
					EVSEID:            update.EVSEID,
					ConnectorID:       update.ConnectorID,
					ConnectorStatus:   event.Current.Status,
					EVSEStatus:        event.Current.EVSEStatus,
					ConnectorType:     event.Current.ConnectorType,
					ReasonCode:        event.Current.ReasonCode,
					VendorID:          event.Current.VendorID,
					VendorDescription: event.Current.VendorDescription,
					StatusTimestamp:   event.Current.StatusTimestamp,
					RecordedAt:        event.Current.UpdatedAt,
				})
				if persistErr != nil {
					log.Printf("StatusNotification persist err for %s: %v\n", stationID, persistErr)
					errPayload := []any{4, uid, "InternalError", "failed to persist connector status", map[string]any{}}
					if sendErr := sendJSON(c, errPayload); sendErr != nil {
						log.Println("write StatusNotification error response err:", sendErr)
						return
					}
					continue
				}

				log.Printf("StatusNotification received from %s: evse=%d connector=%d status=%s (prev=%s)\n",
					stationID, update.EVSEID, update.ConnectorID, event.Current.Status, event.Previous.Status)

				resp := []any{3, uid, map[string]any{}}
				if err := sendJSON(c, resp); err != nil {
					log.Println("write StatusNotification err:", err)
					return
				}

				select {
				case events <- event:
				default:
					log.Printf("status event channel full, dropping event for %s evse=%d connector=%d\n",
						stationID, update.EVSEID, update.ConnectorID)
				}

				if err := repo.UpdateLastSeen(ctx, stationID, now); err != nil {
					log.Printf("update last_seen_at err for %s: %v\n", stationID, err)
				}

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

func parseStatusNotification(frame []any) (registry.StatusUpdate, error) {
	var update registry.StatusUpdate
	if len(frame) < 4 {
		return update, fmt.Errorf("missing payload")
	}

	payload, ok := frame[3].(map[string]any)
	if !ok {
		return update, fmt.Errorf("payload is not an object")
	}

	evseID, err := parsePositiveInt(payload, "evseId")
	if err != nil {
		return update, err
	}
	connectorID, err := parsePositiveInt(payload, "connectorId")
	if err != nil {
		return update, err
	}

	statusRaw, ok := payload["connectorStatus"].(string)
	if !ok || strings.TrimSpace(statusRaw) == "" {
		return update, fmt.Errorf("connectorStatus is required")
	}

	update.EVSEID = evseID
	update.ConnectorID = connectorID
	update.ConnectorStatus = strings.TrimSpace(statusRaw)

	if tsRaw, ok := payload["timestamp"].(string); ok && strings.TrimSpace(tsRaw) != "" {
		ts, err := time.Parse(time.RFC3339, tsRaw)
		if err != nil {
			return update, fmt.Errorf("invalid timestamp: %w", err)
		}
		update.Timestamp = ts
	}

	if evseStatus, ok := payload["evseStatus"].(string); ok {
		update.EVSEStatus = strings.TrimSpace(evseStatus)
	}
	if connectorType, ok := payload["connectorType"].(string); ok {
		update.ConnectorType = strings.TrimSpace(connectorType)
	}
	if reason, ok := payload["reasonCode"].(string); ok {
		update.ReasonCode = strings.TrimSpace(reason)
	}
	if vendorID, ok := payload["vendorId"].(string); ok {
		update.VendorID = strings.TrimSpace(vendorID)
	}
	if vendorDescription, ok := payload["vendorDescription"].(string); ok {
		update.VendorDescription = strings.TrimSpace(vendorDescription)
	}

	return update, nil
}

func parsePositiveInt(payload map[string]any, field string) (int, error) {
	value, ok := payload[field]
	if !ok {
		return 0, fmt.Errorf("%s is required", field)
	}

	num, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be a number", field)
	}
	if math.IsNaN(num) || math.IsInf(num, 0) {
		return 0, fmt.Errorf("%s has invalid numeric value", field)
	}
	if num <= 0 {
		return 0, fmt.Errorf("%s must be positive", field)
	}
	if float64(int(num)) != num {
		return 0, fmt.Errorf("%s must be an integer", field)
	}

	return int(num), nil
}

func main() {
	ctx := context.Background()
	dsn, ok := os.LookupEnv("DATABASE_URL")
	if !ok || strings.TrimSpace(dsn) == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := storage.NewPostgresPool(ctx, dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	repo := storage.NewPostgresStationRepository(pool)
	connectors := registry.New()
	statusEvents := make(chan registry.StatusEvent, 64)

	go func() {
		for event := range statusEvents {
			log.Printf("Status event: station=%s evse=%d connector=%d status=%s prev=%s timestamp=%s recorded=%s\n",
				event.StationID,
				event.Update.EVSEID,
				event.Update.ConnectorID,
				event.Current.Status,
				event.Previous.Status,
				event.Update.Timestamp.Format(time.RFC3339),
				event.RecordedAt.Format(time.RFC3339))
		}
	}()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ocpp/", newOCPPHandler(repo, connectors, statusEvents))

	log.Println("CSMS stub listening on :8080  (ws)  path=/ocpp/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
