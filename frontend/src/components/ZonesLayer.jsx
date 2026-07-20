import { useMemo } from "react";
import { Line } from "@react-three/drei";
import { latLonToVector3, circlePoints } from "../utils/geo";
import { GLOBE_RADIUS } from "./Globe";
import { WATCH_ZONES } from "../data/zones";

const ZONE_ALTITUDE = GLOBE_RADIUS * 1.008;

export default function ZonesLayer() {
  const zoneLines = useMemo(
    () =>
      WATCH_ZONES.map((z) => ({
        name: z.name,
        points: circlePoints(z.centerLat, z.centerLon, z.radiusKm, 96).map((p) =>
          latLonToVector3(p.lat, p.lon, ZONE_ALTITUDE)
        ),
      })),
    []
  );

  return (
    <group>
      {zoneLines.map((zone) => (
        <Line
          key={zone.name}
          points={zone.points}
          color="#ffd23f"
          transparent
          opacity={0.55}
          lineWidth={1.5}
        />
      ))}
    </group>
  );
}
