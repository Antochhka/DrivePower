"""
Simple OCPP-like station simulator with GUI (tkinter).

Dependencies:
    pip install websockets

Run:
    python station_simulator.py

This is a minimal visual tool to connect to your OCPP server, send
BootNotification/Heartbeat/Status/Start/Stop messages, and control sessions
per connector.
"""

import asyncio
import json
import queue
import random
import string
import threading
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Callable, Dict, List, Optional

import tkinter as tk
from tkinter import ttk, messagebox

import websockets


def iso_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def random_tx_id() -> str:
    suffix = "".join(random.choices(string.ascii_uppercase + string.digits, k=6))
    return f"TX-{suffix}"


@dataclass
class ConnectorState:
    connector_id: int
    status: str = "Available"
    current_energy_kwh: float = 0.0
    active_session: bool = False
    transaction_id: Optional[str] = None
    energy_task: Optional[asyncio.Task] = None


class StationSimulator:
    """
    Holds station state, manages websocket connection, and sends OCPP-like messages.
    """

    def __init__(
        self,
        log_fn: Callable[[str], None],
        state_fn: Callable[[List[ConnectorState]], None],
        heartbeat_interval: int = 30,
    ):
        self.log = log_fn
        self.publish_state = state_fn
        self.heartbeat_interval = heartbeat_interval
        self.url: str = ""
        self.station_id: str = ""
        self.connectors: List[ConnectorState] = []
        self.ws: Optional[websockets.WebSocketClientProtocol] = None
        self.loop: Optional[asyncio.AbstractEventLoop] = None
        self.loop_thread: Optional[threading.Thread] = None
        self.connected = False
        self.stop_event = threading.Event()
        self.auto_start = False
        self.sim_speed = 1.0  # multiplier for energy growth

    def set_config(
        self,
        url: str,
        station_id: str,
        connectors_count: int,
        auto_start: bool,
        sim_speed: float,
    ):
        self.url = url
        self.station_id = station_id
        self.auto_start = auto_start
        self.sim_speed = sim_speed
        self.connectors = [
            ConnectorState(connector_id=i + 1) for i in range(connectors_count)
        ]
        self.publish_state(self.connectors)

    def start(self):
        if self.loop_thread and self.loop_thread.is_alive():
            return
        self.stop_event.clear()
        self.loop_thread = threading.Thread(target=self._run_loop, daemon=True)
        self.loop_thread.start()

    def _run_loop(self):
        self.loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self.loop)
        self.loop.run_until_complete(self._connect_and_run())

    async def _connect_and_run(self):
        if not self.url:
            self.log("No URL configured")
            return
        try:
            self.ws = await websockets.connect(self.url)
            self.connected = True
            self.log(f"Connected to {self.url}")
        except Exception as exc:
            self.log(f"Connect failed: {exc}")
            return

        recv_task = asyncio.create_task(self._recv_loop())
        hb_task = asyncio.create_task(self._heartbeat_loop())

        # Boot + initial status
        await self._send_boot()
        await self._send_all_status("Available")

        if self.auto_start and self.connectors:
            await self._start_session(self.connectors[0].connector_id)

        try:
            await asyncio.wait([recv_task, hb_task], return_when=asyncio.FIRST_COMPLETED)
        finally:
            recv_task.cancel()
            hb_task.cancel()
            await self._cleanup()

    async def _cleanup(self):
        for c in self.connectors:
            if c.energy_task:
                c.energy_task.cancel()
                c.energy_task = None
        if self.ws:
            try:
                await self.ws.close()
            except Exception:
                pass
        self.connected = False
        self.log("Disconnected")

    def disconnect(self):
        if self.loop:
            self.loop.call_soon_threadsafe(self.loop.stop)
        self.stop_event.set()

    async def _recv_loop(self):
        try:
            async for msg in self.ws:
                self.log(f"RECV: {msg}")
                self._handle_incoming(msg)
        except Exception as exc:
            self.log(f"Receive error: {exc}")

    def _handle_incoming(self, msg: str):
        try:
            data = json.loads(msg)
        except json.JSONDecodeError:
            return
        if not isinstance(data, dict):
            return
        msg_type = data.get("messageType")
        if msg_type == "RemoteStartTransaction":
            connector_id = int(data.get("connectorId", 1))
            self.run_coro(self._start_session(connector_id))
        elif msg_type == "RemoteStopTransaction":
            connector_id = int(data.get("connectorId", 1))
            self.run_coro(self._stop_session(connector_id))

    async def _heartbeat_loop(self):
        while self.connected and self.ws and not self.stop_event.is_set():
            await asyncio.sleep(self.heartbeat_interval)
            if self.connected:
                frame = [
                    2,
                    random_tx_id(),
                    "Heartbeat",
                    {"timestamp": iso_now()},
                ]
                await self._send_frame(frame)
                self.log(f"SEND: {frame}")

    async def _send_boot(self):
        frame = [
            2,
            random_tx_id(),
            "BootNotification",
            {
                "stationId": self.station_id,
                "chargePointVendor": "SimVendor",
                "chargePointModel": "SimModel",
            },
        ]
        await self._send_frame(frame)
        self.log(f"SEND: {frame}")

    async def _send_all_status(self, status: str):
        for c in self.connectors:
            frame = [
                2,
                random_tx_id(),
                "StatusNotification",
                {
                    "stationId": self.station_id,
                    "connectorId": c.connector_id,
                    "status": status,
                    "timestamp": iso_now(),
                },
            ]
            await self._send_frame(frame)
            self.log(f"SEND: {frame}")

    async def _send_frame(self, obj: Any):
        if not self.ws or not self.connected:
            self.log("Not connected")
            return
        await self.ws.send(json.dumps(obj))

    def run_coro(self, coro: asyncio.Future):
        if not self.loop:
            self.log("Event loop not running")
            return
        asyncio.run_coroutine_threadsafe(coro, self.loop)

    async def _start_session(self, connector_id: int):
        connector = self._get_connector(connector_id)
        if not connector or connector.active_session:
            return
        tx_id = random_tx_id()
        connector.active_session = True
        connector.transaction_id = tx_id
        connector.status = "Charging"
        connector.current_energy_kwh = 0.0
        self.publish_state(self.connectors)

        frame = [
            2,
            random_tx_id(),
            "StartTransaction",
            {
                "stationId": self.station_id,
                "connectorId": connector_id,
                "transactionId": tx_id,
                "idTag": "TAG-001",
                "meterStart": int(connector.current_energy_kwh * 1000),
                "timestamp": iso_now(),
            },
        ]
        await self._send_frame(frame)
        self.log(f"SEND: {frame}")

        connector.energy_task = asyncio.create_task(self._energy_simulator(connector))

    async def _stop_session(self, connector_id: int):
        connector = self._get_connector(connector_id)
        if not connector or not connector.active_session or not connector.transaction_id:
            return
        if connector.energy_task:
            connector.energy_task.cancel()
            connector.energy_task = None

        payload_body = {
            "stationId": self.station_id,
            "connectorId": connector_id,
            "transactionId": connector.transaction_id,
            "idTag": "TAG-001",
            "meterStop": int(connector.current_energy_kwh * 1000),
            "timestamp": iso_now(),
            "reason": "Local",
        }
        connector.active_session = False
        connector.transaction_id = None
        connector.status = "Available"
        self.publish_state(self.connectors)

        frame = [2, random_tx_id(), "StopTransaction", payload_body]
        await self._send_frame(frame)
        self.log(f"SEND: {frame}")

    async def _energy_simulator(self, connector: ConnectorState):
        try:
            while connector.active_session:
                await asyncio.sleep(2 / self.sim_speed)
                connector.current_energy_kwh += 0.1 * self.sim_speed
                self.publish_state(self.connectors)
                # send meter values periodically
                if connector.transaction_id:
                    frame = [
                        2,
                        random_tx_id(),
                        "MeterValues",
                        {
                            "stationId": self.station_id,
                            "connectorId": connector.connector_id,
                            "transactionId": connector.transaction_id,
                            "meterValue": round(connector.current_energy_kwh, 3),
                            "timestamp": iso_now(),
                        },
                    ]
                    await self._send_frame(frame)
                    self.log(f"SEND: {frame}")
        except asyncio.CancelledError:
            pass

    def _get_connector(self, connector_id: int) -> Optional[ConnectorState]:
        for c in self.connectors:
            if c.connector_id == connector_id:
                return c
        return None


class GUIMainWindow:
    """
    Builds tkinter UI and bridges user actions to StationSimulator.
    """

    def __init__(self, root: tk.Tk):
        self.root = root
        self.root.title("OCPP Station Simulator")
        self.log_queue: queue.Queue[str] = queue.Queue()
        self.state_data: List[ConnectorState] = []

        self.simulator = StationSimulator(
            log_fn=self.enqueue_log, state_fn=self.update_state
        )

        self._build_layout()
        self._start_pollers()

    def enqueue_log(self, msg: str):
        timestamped = f"[{datetime.now().strftime('%H:%M:%S')}] {msg}"
        self.log_queue.put(timestamped)

    def update_state(self, connectors: List[ConnectorState]):
        self.state_data = connectors

    def _start_pollers(self):
        self.root.after(200, self._drain_logs)
        self.root.after(500, self._refresh_state)

    def _drain_logs(self):
        while not self.log_queue.empty():
            msg = self.log_queue.get()
            self.log_text.insert(tk.END, msg + "\n")
            self.log_text.see(tk.END)
        self.root.after(200, self._drain_logs)

    def _refresh_state(self):
        for i, connector in enumerate(self.state_data):
            if i >= len(self.state_rows):
                break
            row = self.state_rows[i]
            row["status_var"].set(connector.status)
            row["energy_var"].set(f"{connector.current_energy_kwh:.2f}")
            row["tx_var"].set(connector.transaction_id or "")
        self.root.after(500, self._refresh_state)

    def _build_layout(self):
        top = ttk.LabelFrame(self.root, text="Configuration")
        top.pack(fill="x", padx=8, pady=4)

        ttk.Label(top, text="CSMS URL:").grid(row=0, column=0, sticky="w")
        self.url_var = tk.StringVar(
            value="ws://ocpp-server:8081/ocpp/ws?station_id=CS-001"
        )
        ttk.Entry(top, textvariable=self.url_var, width=50).grid(
            row=0, column=1, sticky="we", padx=4, pady=2
        )

        ttk.Label(top, text="Station ID:").grid(row=1, column=0, sticky="w")
        self.station_var = tk.StringVar(value="CS-001")
        ttk.Entry(top, textvariable=self.station_var, width=20).grid(
            row=1, column=1, sticky="w", padx=4, pady=2
        )

        ttk.Label(top, text="Connectors:").grid(row=2, column=0, sticky="w")
        self.connectors_var = tk.IntVar(value=1)
        ttk.Spinbox(top, from_=1, to=4, textvariable=self.connectors_var, width=5).grid(
            row=2, column=1, sticky="w", padx=4, pady=2
        )

        ttk.Label(top, text="Heartbeat (s):").grid(row=3, column=0, sticky="w")
        self.hb_var = tk.IntVar(value=30)
        ttk.Spinbox(top, from_=5, to=120, textvariable=self.hb_var, width=5).grid(
            row=3, column=1, sticky="w", padx=4, pady=2
        )

        ttk.Label(top, text="Sim speed:").grid(row=4, column=0, sticky="w")
        self.speed_var = tk.DoubleVar(value=1.0)
        ttk.Combobox(top, textvariable=self.speed_var, values=[0.5, 1.0, 2.0, 4.0], width=5).grid(
            row=4, column=1, sticky="w", padx=4, pady=2
        )

        self.auto_start_var = tk.BooleanVar(value=False)
        ttk.Checkbutton(top, text="Auto-start on connect", variable=self.auto_start_var).grid(
            row=5, column=1, sticky="w", padx=4, pady=2
        )

        btn_frame = ttk.Frame(top)
        btn_frame.grid(row=0, column=2, rowspan=3, padx=6)
        ttk.Button(btn_frame, text="Connect", command=self.on_connect).pack(
            fill="x", pady=2
        )
        ttk.Button(btn_frame, text="Disconnect", command=self.on_disconnect).pack(
            fill="x", pady=2
        )

        # Connectors section
        con_frame = ttk.LabelFrame(self.root, text="Connectors")
        con_frame.pack(fill="x", padx=8, pady=4)

        header = ttk.Frame(con_frame)
        header.pack(fill="x")
        for idx, text in enumerate(["ID", "Status", "Energy (kWh)", "TxID", "Actions"]):
            ttk.Label(header, text=text, width=15).grid(row=0, column=idx, padx=2)

        self.state_rows: List[Dict[str, Any]] = []
        for i in range(4):
            row_frame = ttk.Frame(con_frame)
            row_frame.pack(fill="x")
            status_var = tk.StringVar(value="-")
            energy_var = tk.StringVar(value="0.00")
            tx_var = tk.StringVar(value="")
            ttk.Label(row_frame, text=str(i + 1), width=5).grid(row=0, column=0)
            ttk.Label(row_frame, textvariable=status_var, width=12).grid(
                row=0, column=1
            )
            ttk.Label(row_frame, textvariable=energy_var, width=12).grid(
                row=0, column=2
            )
            ttk.Label(row_frame, textvariable=tx_var, width=20).grid(row=0, column=3)
            btn_start = ttk.Button(
                row_frame,
                text="Start session",
                command=lambda cid=i + 1: self.on_start(cid),
            )
            btn_stop = ttk.Button(
                row_frame, text="Stop session", command=lambda cid=i + 1: self.on_stop(cid)
            )
            btn_start.grid(row=0, column=4, padx=2)
            btn_stop.grid(row=0, column=5, padx=2)
            self.state_rows.append(
                {
                    "status_var": status_var,
                    "energy_var": energy_var,
                    "tx_var": tx_var,
                    "frame": row_frame,
                }
            )

        # Log area
        log_frame = ttk.LabelFrame(self.root, text="Log")
        log_frame.pack(fill="both", expand=True, padx=8, pady=4)
        self.log_text = tk.Text(log_frame, height=10, wrap="word")
        self.log_text.pack(fill="both", expand=True)

    def on_connect(self):
        url = self.url_var.get().strip()
        station_id = self.station_var.get().strip()
        connectors = self.connectors_var.get()
        hb = self.hb_var.get()
        speed = float(self.speed_var.get())
        if not url or not station_id:
            messagebox.showerror("Error", "URL and Station ID are required")
            return
        # Normalize URL to include station_id if missing
        if "station_id=" not in url:
            sep = "&" if "?" in url else "?"
            url = f"{url}{sep}station_id={station_id}"
            self.url_var.set(url)

        self.simulator.heartbeat_interval = hb
        self.simulator.set_config(
            url=url,
            station_id=station_id,
            connectors_count=connectors,
            auto_start=self.auto_start_var.get(),
            sim_speed=speed,
        )
        self.simulator.start()
        self.enqueue_log("Connecting...")

    def on_disconnect(self):
        self.simulator.disconnect()
        self.enqueue_log("Disconnect requested")

    def on_start(self, connector_id: int):
        self.simulator.run_coro(self.simulator._start_session(connector_id))

    def on_stop(self, connector_id: int):
        self.simulator.run_coro(self.simulator._stop_session(connector_id))


def main():
    root = tk.Tk()
    app = GUIMainWindow(root)
    root.mainloop()


if __name__ == "__main__":
    main()
