# Orbit — Live 3D Flight Tracker

A WebGL globe (Three.js/react-three-fiber) showing real-world flights in
real time. A Go backend polls airplanes.live's community ADS-B feed and
fans
updates out to every connected browser over a single shared WebSocket
connection, so you get live data without every visitor hammering the
upstream API individually.

## Structure
- `backend/` — Go WebSocket server + flight/ship/satellite/weather tracking
- `frontend/` — React + Vite + Three.js globe UI

## Run locally

**Backend** (needs Go 1.21+):
```
cd backend
go mod tidy
go run .
```
Runs on `http://localhost:8080`, WebSocket at `ws://localhost:8080/ws`.

**Frontend** (needs Node 18+):
```
cd frontend
npm install
cp .env.example .env
npm run dev
```
Opens at `http://localhost:5173`. It'll connect to the local backend automatically.

## Deploy
- **Backend** → Railway or Fly.io (both auto-detect Go, give you a public
  `https://...` URL — use `wss://` for the WebSocket URL in the frontend env).
- **Frontend** → Vercel or Netlify. Set `VITE_WS_URL` env var to your
  deployed backend's WebSocket URL before building.
