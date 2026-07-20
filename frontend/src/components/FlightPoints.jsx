import { useEffect, useMemo, useRef, useState } from "react";
import { useFrame } from "@react-three/fiber";
import { Line } from "@react-three/drei";
import * as THREE from "three";
import { latLonToVector3, headingToTangent, projectPosition } from "../utils/geo";
import { createPlaneIconGeometry } from "../utils/iconGeometries";
import { GLOBE_RADIUS } from "./Globe";

const MAX_FLIGHTS = 4000;
const DOT_ALTITUDE = GLOBE_RADIUS * 1.02;

// How long (ms) we smoothly interpolate a dot from its old position to its
// newly-reported one, so movement between backend polls looks fluid.
const LERP_MS = 2500;

// Short comet-tail ghosting (purely decorative, sub-second timescale).
const TRAIL_LAYERS = 4;
const TRAIL_SAMPLE_MS = 180;

// Real flown-path history (informational, one point per actual poll ~12s
// apart) — kept for every aircraft but only rendered for the selected one,
// so we're not drawing thousands of line objects at once.
const MAX_PATH_POINTS = 20;

const UP = new THREE.Vector3(0, 1, 0);
const FACING = new THREE.Vector3(0, 0, 1);

export default function FlightPoints({ flights, selectedIcao, onSelect }) {
  const mainMeshRef = useRef();
  const riskRingRef = useRef();
  const trailRefs = useRef([]);
  const dummy = useMemo(() => new THREE.Object3D(), []);
  const planeIconGeometry = useMemo(() => createPlaneIconGeometry(), []);

  const trackRef = useRef(new Map());
  const orderRef = useRef([]);
  const sampleAccumRef = useRef(0);

  const [selectedPath, setSelectedPath] = useState([]);
  const [predictedPoint, setPredictedPoint] = useState(null);

  // How far ahead (seconds) we project the dashed "predicted position" line.
  const PREDICTION_SECONDS = 600; // 10 minutes

  useEffect(() => {
    const now = performance.now();
    const track = trackRef.current;
    const seen = new Set();

    for (const f of flights) {
      if (!f.icao24) continue;
      seen.add(f.icao24);
      const target = latLonToVector3(f.lat, f.lon, DOT_ALTITUDE);
      const forward = headingToTangent(f.lat, f.lon, f.heading || 0, 1);
      const existing = track.get(f.icao24);

      if (existing) {
        existing.from.copy(existing.current);
        existing.to.copy(target);
        existing.startedAt = now;
        existing.forward.copy(forward);
        existing.data = f;
        existing.path.push(target.clone());
        if (existing.path.length > MAX_PATH_POINTS) existing.path.shift();
      } else {
        track.set(f.icao24, {
          current: target.clone(),
          from: target.clone(),
          to: target.clone(),
          forward,
          startedAt: now,
          data: f,
          history: Array.from({ length: TRAIL_LAYERS }, () => target.clone()),
          path: [target.clone()],
        });
      }
    }

    for (const key of Array.from(track.keys())) {
      if (!seen.has(key)) track.delete(key);
    }

    orderRef.current = Array.from(track.keys()).slice(0, MAX_FLIGHTS);

    // Refresh the visible path and ETA projection for whichever flight is
    // currently selected.
    if (selectedIcao) {
      const entry = track.get(selectedIcao);
      if (entry) {
        setSelectedPath(entry.path.map((v) => v.clone()));
        const f = entry.data;
        const projected = projectPosition(
          f.lat, f.lon, f.heading || 0, f.velocity || 0, PREDICTION_SECONDS, DOT_ALTITUDE
        );
        setPredictedPoint(projected.point);
      } else {
        setSelectedPath([]);
        setPredictedPoint(null);
      }
    } else {
      setSelectedPath([]);
      setPredictedPoint(null);
    }
  }, [flights, selectedIcao]);

  useFrame((_, delta) => {
    const mesh = mainMeshRef.current;
    if (!mesh) return;
    const now = performance.now();
    const track = trackRef.current;
    const order = orderRef.current;

    sampleAccumRef.current += delta * 1000;
    const shouldSample = sampleAccumRef.current >= TRAIL_SAMPLE_MS;
    if (shouldSample) sampleAccumRef.current = 0;

    for (let i = 0; i < order.length; i++) {
      const entry = track.get(order[i]);
      if (!entry) continue;

      const t = Math.min(1, (now - entry.startedAt) / LERP_MS);
      entry.current.lerpVectors(entry.from, entry.to, t);

      if (shouldSample) {
        for (let h = entry.history.length - 1; h > 0; h--) {
          entry.history[h].copy(entry.history[h - 1]);
        }
        entry.history[0].copy(entry.current);
      }

      dummy.position.copy(entry.current);
      // Orient the arrow's apex along the aircraft's actual heading, in
      // the local tangent plane at its position — this is what makes
      // each marker show which way the flight is actually going, not
      // just where it currently is.
      dummy.quaternion.setFromUnitVectors(UP, entry.forward);
      dummy.scale.setScalar(1);
      dummy.updateMatrix();
      mesh.setMatrixAt(i, dummy.matrix);
    }
    mesh.count = order.length;
    mesh.instanceMatrix.needsUpdate = true;

    const ringMesh = riskRingRef.current;
    if (ringMesh) {
      let ringCount = 0;
      const pulse = 1 + 0.35 * Math.sin(now / 300);
      for (let i = 0; i < order.length; i++) {
        const entry = track.get(order[i]);
        if (!entry || (entry.data.riskScore || 0) < 50) continue;

        dummy.position.copy(entry.current);
        dummy.quaternion.setFromUnitVectors(FACING, entry.current.clone().normalize());
        dummy.scale.setScalar(pulse);
        dummy.updateMatrix();
        ringMesh.setMatrixAt(ringCount, dummy.matrix);
        ringCount++;
      }
      ringMesh.count = ringCount;
      ringMesh.instanceMatrix.needsUpdate = true;
    }

    for (let layer = 0; layer < TRAIL_LAYERS; layer++) {
      const trailMesh = trailRefs.current[layer];
      if (!trailMesh) continue;
      const scale = Math.max(0.3, 1 - (layer + 1) / (TRAIL_LAYERS + 1));

      for (let i = 0; i < order.length; i++) {
        const entry = track.get(order[i]);
        if (!entry) continue;
        dummy.position.copy(entry.history[layer]);
        dummy.quaternion.identity();
        dummy.scale.setScalar(scale);
        dummy.updateMatrix();
        trailMesh.setMatrixAt(i, dummy.matrix);
      }
      trailMesh.count = order.length;
      trailMesh.instanceMatrix.needsUpdate = true;
    }
  });

  const handleClick = (event) => {
    event.stopPropagation();
    const key = orderRef.current[event.instanceId];
    const entry = trackRef.current.get(key);
    if (entry) onSelect(entry.data);
  };

  return (
    <group>
      {Array.from({ length: TRAIL_LAYERS }).map((_, layer) => (
        <instancedMesh
          key={layer}
          ref={(el) => (trailRefs.current[layer] = el)}
          args={[null, null, MAX_FLIGHTS]}
        >
          <sphereGeometry args={[0.01, 6, 6]} />
          <meshBasicMaterial
            color="#ff6a3d"
            transparent
            opacity={0.2 - layer * 0.04}
            toneMapped={false}
            blending={THREE.AdditiveBlending}
            depthWrite={false}
          />
        </instancedMesh>
      ))}

      {/* Aircraft silhouette (dart shape) oriented to real heading */}
      <instancedMesh ref={mainMeshRef} args={[null, null, MAX_FLIGHTS]} onClick={handleClick}>
        <primitive object={planeIconGeometry} attach="geometry" />
        <meshBasicMaterial color="#ff8c5a" toneMapped={false} side={THREE.DoubleSide} />
      </instancedMesh>

      {/* Pulsing red ring — visible flag for flights with elevated
          correlated risk (zone + weather + anomaly), no click needed */}
      <instancedMesh ref={riskRingRef} args={[null, null, MAX_FLIGHTS]}>
        <ringGeometry args={[0.02, 0.026, 20]} />
        <meshBasicMaterial
          color="#ef4444"
          transparent
          opacity={0.75}
          side={THREE.DoubleSide}
          depthWrite={false}
        />
      </instancedMesh>

      {selectedPath.length > 1 && (
        <Line
          points={selectedPath}
          color="#38bdf8"
          lineWidth={2}
          transparent
          opacity={0.85}
        />
      )}

      {predictedPoint && selectedPath.length > 0 && (
        <Line
          points={[selectedPath[selectedPath.length - 1], predictedPoint]}
          color="#facc15"
          lineWidth={1.5}
          dashed
          dashSize={0.03}
          gapSize={0.02}
          transparent
          opacity={0.7}
        />
      )}
    </group>
  );
}
