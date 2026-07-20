import { useEffect, useMemo, useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { latLonToVector3, headingToTangent } from "../utils/geo";
import { createShipIconGeometry } from "../utils/iconGeometries";
import { GLOBE_RADIUS } from "./Globe";

const MAX_SHIPS = 500;
const SHIP_ALTITUDE = GLOBE_RADIUS * 1.006; // just above the surface, below flight altitude
const LERP_MS = 4000; // ships move slowly - a gentler interpolation than flights
const UP = new THREE.Vector3(0, 1, 0);

export default function ShipsLayer({ ships, onSelect }) {
  const meshRef = useRef();
  const dummy = useMemo(() => new THREE.Object3D(), []);
  const shipIconGeometry = useMemo(() => createShipIconGeometry(), []);

  const trackRef = useRef(new Map());
  const orderRef = useRef([]);

  useEffect(() => {
    const now = performance.now();
    const track = trackRef.current;
    const seen = new Set();

    for (const s of ships) {
      if (!s.mmsi) continue;
      seen.add(s.mmsi);
      const target = latLonToVector3(s.lat, s.lon, SHIP_ALTITUDE);
      const forward = headingToTangent(s.lat, s.lon, s.courseDeg || 0, 1);
      const existing = track.get(s.mmsi);

      if (existing) {
        existing.from.copy(existing.current);
        existing.to.copy(target);
        existing.startedAt = now;
        existing.forward.copy(forward);
        existing.data = s;
      } else {
        track.set(s.mmsi, {
          current: target.clone(),
          from: target.clone(),
          to: target.clone(),
          forward,
          startedAt: now,
          data: s,
        });
      }
    }

    for (const key of Array.from(track.keys())) {
      if (!seen.has(key)) track.delete(key);
    }
    orderRef.current = Array.from(track.keys()).slice(0, MAX_SHIPS);
  }, [ships]);

  useFrame(() => {
    const mesh = meshRef.current;
    if (!mesh) return;
    const now = performance.now();
    const track = trackRef.current;
    const order = orderRef.current;

    for (let i = 0; i < order.length; i++) {
      const entry = track.get(order[i]);
      if (!entry) continue;
      const t = Math.min(1, (now - entry.startedAt) / LERP_MS);
      entry.current.lerpVectors(entry.from, entry.to, t);

      dummy.position.copy(entry.current);
      dummy.quaternion.setFromUnitVectors(UP, entry.forward);
      dummy.scale.setScalar(1);
      dummy.updateMatrix();
      mesh.setMatrixAt(i, dummy.matrix);
    }
    mesh.count = order.length;
    mesh.instanceMatrix.needsUpdate = true;
  });

  const handleClick = (event) => {
    event.stopPropagation();
    const key = orderRef.current[event.instanceId];
    const entry = trackRef.current.get(key);
    if (entry) onSelect(entry.data);
  };

  return (
    <instancedMesh ref={meshRef} args={[null, null, MAX_SHIPS]} onClick={handleClick}>
      <primitive object={shipIconGeometry} attach="geometry" />
      <meshBasicMaterial color="#34d399" toneMapped={false} side={THREE.DoubleSide} />
    </instancedMesh>
  );
}
