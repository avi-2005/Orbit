import { useEffect, useRef, useState } from "react";

const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws";

/**
 * Connects to the Orbit backend WebSocket and keeps the latest flight
 * snapshot in state. Reconnects automatically with backoff if the
 * connection drops, since this is meant to run unattended in a browser
 * tab for a while.
 */
export function useFlightSocket() {
  const [flights, setFlights] = useState([]);
  const [insights, setInsights] = useState([]);
  const [satellites, setSatellites] = useState([]);
  const [weather, setWeather] = useState([]);
  const [status, setStatus] = useState("connecting");
  const retryDelay = useRef(1000);

  useEffect(() => {
    let socket;
    let cancelled = false;

    function connect() {
      socket = new WebSocket(WS_URL);

      socket.onopen = () => {
        if (cancelled) return;
        setStatus("connected");
        retryDelay.current = 1000;
      };

      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "flights") {
            setFlights(msg.flights || []);
            if (msg.insights && msg.insights.length > 0) {
              setInsights((prev) => [...msg.insights, ...prev].slice(0, 30));
            }
          } else if (msg.type === "satellites") {
            setSatellites(msg.satellites || []);
          } else if (msg.type === "weather") {
            setWeather(msg.weather || []);
          }
        } catch (err) {
          console.error("failed to parse ws message", err);
        }
      };

      socket.onclose = () => {
        if (cancelled) return;
        setStatus("disconnected");
        setTimeout(connect, retryDelay.current);
        retryDelay.current = Math.min(retryDelay.current * 1.5, 15000);
      };

      socket.onerror = () => {
        socket.close();
      };
    }

    connect();
    return () => {
      cancelled = true;
      socket && socket.close();
    };
  }, []);

  return { flights, insights, satellites, weather, status };
}
