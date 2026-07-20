// Mirrors backend/zones.go — kept as a small duplicated constant rather
// than fetched from the server, since it's static reference data.
export const WATCH_ZONES = [
  { name: "Black Sea / Ukraine Airspace Advisory", centerLat: 49.0, centerLon: 32.0, radiusKm: 500 },
  { name: "Red Sea Maritime & Air Risk Zone", centerLat: 15.5, centerLon: 41.5, radiusKm: 450 },
  { name: "North Korea Restricted Airspace", centerLat: 40.0, centerLon: 127.5, radiusKm: 300 },
  { name: "Strait of Hormuz Oil Chokepoint", centerLat: 26.5, centerLon: 56.3, radiusKm: 200 },
];
