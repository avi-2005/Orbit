import { useMemo } from "react";
import * as THREE from "three";
import { latLonToVector3 } from "../utils/geo";
import { GLOBE_RADIUS } from "./Globe";

const EARTH_RADIUS_KM = 6378;
const MAX_SATELLITES = 250;

// A single satellite: small body + two solar-panel wings — the classic,
// instantly-recognizable satellite silhouette. Rendered as real meshes
// (not instanced) since satellite counts are small (~100-200), so there's
// no performance reason to force it through the instancing path flights
// need at 4,000+.
function SatelliteMarker({ satellite, onSelect }) {
  const radius = GLOBE_RADIUS * (1 + satellite.altitudeKm / EARTH_RADIUS_KM);
  const position = useMemo(
    () => latLonToVector3(satellite.lat, satellite.lon, radius),
    [satellite.lat, satellite.lon, radius]
  );

  return (
    <group
      position={position}
      onClick={(e) => {
        e.stopPropagation();
        onSelect(satellite);
      }}
    >
      <mesh>
        <boxGeometry args={[0.012, 0.012, 0.012]} />
        <meshBasicMaterial color="#7dd3fc" toneMapped={false} />
      </mesh>
      <mesh position={[-0.02, 0, 0]}>
        <boxGeometry args={[0.022, 0.009, 0.002]} />
        <meshBasicMaterial color="#38bdf8" toneMapped={false} />
      </mesh>
      <mesh position={[0.02, 0, 0]}>
        <boxGeometry args={[0.022, 0.009, 0.002]} />
        <meshBasicMaterial color="#38bdf8" toneMapped={false} />
      </mesh>
    </group>
  );
}

export default function SatellitesLayer({ satellites, onSelect }) {
  const visible = satellites.slice(0, MAX_SATELLITES);

  return (
    <group>
      {visible.map((sat) => (
        <SatelliteMarker key={sat.name} satellite={sat} onSelect={onSelect} />
      ))}
    </group>
  );
}
