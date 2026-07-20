import { useMemo, useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { latLonToVector3 } from "../utils/geo";
import { GLOBE_RADIUS } from "./Globe";

const MAX_POINTS = 250;
const WEATHER_ALTITUDE = GLOBE_RADIUS * 1.03;
const FACING = new THREE.Vector3(0, 0, 1);

// Classify each grid point into an actual phenomenon rather than a single
// generic "intensity" — this is what makes it read as "weather happening"
// instead of an abstract heatmap.
function classify(w) {
  if (w.temperatureC >= 38) return "heatwave";
  if (w.precipitationMm > 0.1 || w.windSpeedKmh > 35) return "storm";
  return "calm";
}

export default function WeatherLayer({ weather, onSelect }) {
  const meshRef = useRef();
  const dummy = useMemo(() => new THREE.Object3D(), []);
  const color = useMemo(() => new THREE.Color(), []);
  const orderRef = useRef([]);

  useFrame(({ clock }) => {
    const mesh = meshRef.current;
    if (!mesh) return;
    const count = Math.min(weather.length, MAX_POINTS);
    orderRef.current = weather.slice(0, count);
    const t = clock.getElapsedTime();

    for (let i = 0; i < count; i++) {
      const w = weather[i];
      const kind = classify(w);

      let baseColor, intensity;
      if (kind === "heatwave") {
        baseColor = [1.0, 0.45, 0.2]; // orange/red
        intensity = Math.min(1, (w.temperatureC - 38) / 12);
      } else if (kind === "storm") {
        baseColor = [0.6, 0.75, 1.0]; // storm blue-grey
        intensity = Math.min(1, w.precipitationMm / 5 + w.windSpeedKmh / 100);
      } else {
        baseColor = [0.4, 0.55, 0.75];
        intensity = 0.05; // near-invisible ambient marker for calm cells
      }

      dummy.position.copy(latLonToVector3(w.lat, w.lon, WEATHER_ALTITUDE));
      dummy.quaternion.setFromUnitVectors(FACING, dummy.position.clone().normalize());

      // Storms get a slow pulsing scale to read as "active weather", not
      // a static marker — the closest we can do to "moving" without a
      // full animated radar-tile overlay.
      const pulse = kind === "calm" ? 1 : 1 + 0.15 * Math.sin(t * 1.5 + i);
      dummy.scale.setScalar((0.1 + intensity * 1.1) * pulse);
      dummy.updateMatrix();
      mesh.setMatrixAt(i, dummy.matrix);

      color.setRGB(...baseColor).multiplyScalar(0.15 + intensity * 0.85);
      mesh.setColorAt(i, color);
    }
    mesh.count = count;
    mesh.instanceMatrix.needsUpdate = true;
    if (mesh.instanceColor) mesh.instanceColor.needsUpdate = true;
  });

  const handleClick = (event) => {
    event.stopPropagation();
    const point = orderRef.current[event.instanceId];
    if (point) onSelect(point);
  };

  return (
    <instancedMesh ref={meshRef} args={[null, null, MAX_POINTS]} onClick={handleClick}>
      <circleGeometry args={[0.04, 16]} />
      <meshBasicMaterial
        transparent
        opacity={0.4}
        blending={THREE.AdditiveBlending}
        depthWrite={false}
        side={THREE.DoubleSide}
      />
    </instancedMesh>
  );
}
