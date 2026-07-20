import * as THREE from "three";

/**
 * A simple dart/paper-airplane silhouette, nose pointing along +Y so it
 * lines up with the heading-alignment convention used in FlightPoints
 * (the instance's local up-axis gets rotated to point along the real
 * compass heading). Extruded with a little thickness so it stays visible
 * from any camera angle instead of vanishing edge-on like a flat sprite
 * would.
 */
export function createPlaneIconGeometry() {
  const shape = new THREE.Shape();
  shape.moveTo(0, 0.024); // nose
  shape.lineTo(0.021, -0.008); // right wingtip
  shape.lineTo(0.005, -0.006);
  shape.lineTo(0, -0.018); // tail
  shape.lineTo(-0.005, -0.006);
  shape.lineTo(-0.021, -0.008); // left wingtip
  shape.closePath();

  const geometry = new THREE.ExtrudeGeometry(shape, {
    depth: 0.007,
    bevelEnabled: false,
  });
  geometry.center();
  return geometry;
}

/**
 * A boat/hull silhouette — pointed bow, flat stern — nose (bow) pointing
 * along +Y to match the same heading-alignment convention as the plane
 * icon, just driven by AIS course-over-ground instead of aircraft heading.
 */
export function createShipIconGeometry() {
  const shape = new THREE.Shape();
  shape.moveTo(0, 0.02); // bow
  shape.lineTo(0.009, -0.006);
  shape.lineTo(0.009, -0.014); // stern right
  shape.lineTo(-0.009, -0.014); // stern left
  shape.lineTo(-0.009, -0.006);
  shape.closePath();

  const geometry = new THREE.ExtrudeGeometry(shape, {
    depth: 0.006,
    bevelEnabled: false,
  });
  geometry.center();
  return geometry;
}
