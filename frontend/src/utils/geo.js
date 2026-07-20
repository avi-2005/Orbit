import * as THREE from "three";

/**
 * Converts geographic coordinates to a position on a 3D sphere.
 * Standard spherical-to-cartesian conversion, oriented so that
 * (lat=0, lon=0) sits on the +Z axis facing the default camera.
 */
export function latLonToVector3(lat, lon, radius) {
  const phi = (90 - lat) * (Math.PI / 180);
  const theta = (lon + 180) * (Math.PI / 180);

  const x = -radius * Math.sin(phi) * Math.cos(theta);
  const y = radius * Math.cos(phi);
  const z = radius * Math.sin(phi) * Math.sin(theta);

  return new THREE.Vector3(x, y, z);
}

/**
 * Returns a unit vector tangent to the sphere at (lat, lon), pointing in
 * the compass direction given by headingDeg (0 = north, 90 = east).
 * Computed via finite differences — offset slightly in lat and lon to
 * approximate the local north/east tangent directions, then blend them
 * by the heading angle. Precise enough for orienting a small direction
 * marker; not meant for navigation-grade accuracy.
 */
export function headingToTangent(lat, lon, headingDeg, radius) {
  const eps = 0.05;
  const base = latLonToVector3(lat, lon, radius);
  const north = latLonToVector3(Math.min(89.9, lat + eps), lon, radius)
    .sub(base)
    .normalize();
  const east = latLonToVector3(lat, lon + eps, radius).sub(base).normalize();

  const rad = (headingDeg * Math.PI) / 180;
  return north
    .multiplyScalar(Math.cos(rad))
    .add(east.multiplyScalar(Math.sin(rad)))
    .normalize();
}

/**
 * Projects where an aircraft will be after `seconds` more flight time,
 * assuming it holds its current heading and ground speed — the same
 * great-circle "destination point" formula used for the watch-zone
 * circles, just driven by real heading/speed instead of a fixed bearing.
 * This is a straight-line dead-reckoning estimate, not a real flight-plan
 * lookup — worth being upfront about that distinction.
 */
export function projectPosition(lat, lon, headingDeg, speedMs, seconds, radius) {
  const earthRadiusM = 6371000;
  const angularDistance = (speedMs * seconds) / earthRadiusM;
  const bearing = (headingDeg * Math.PI) / 180;
  const latRad = (lat * Math.PI) / 180;
  const lonRad = (lon * Math.PI) / 180;

  const lat2 = Math.asin(
    Math.sin(latRad) * Math.cos(angularDistance) +
      Math.cos(latRad) * Math.sin(angularDistance) * Math.cos(bearing)
  );
  const lon2 =
    lonRad +
    Math.atan2(
      Math.sin(bearing) * Math.sin(angularDistance) * Math.cos(latRad),
      Math.cos(angularDistance) - Math.sin(latRad) * Math.sin(lat2)
    );

  const outLat = (lat2 * 180) / Math.PI;
  const outLon = (((lon2 * 180) / Math.PI + 540) % 360) - 180;

  return { lat: outLat, lon: outLon, point: latLonToVector3(outLat, outLon, radius) };
}

/**
 * Generates points around a circle of a given radius (km) centered on a
 * lat/lon, using the standard "destination point given bearing + distance"
 * spherical formula. Used to draw watch-zone outlines on the globe.
 */
export function circlePoints(centerLat, centerLon, radiusKm, segments = 80) {
  const EARTH_RADIUS_KM = 6371;
  const angularRadius = radiusKm / EARTH_RADIUS_KM;
  const latRad = (centerLat * Math.PI) / 180;
  const lonRad = (centerLon * Math.PI) / 180;

  const points = [];
  for (let i = 0; i <= segments; i++) {
    const bearing = (i / segments) * 2 * Math.PI;
    const lat2 = Math.asin(
      Math.sin(latRad) * Math.cos(angularRadius) +
        Math.cos(latRad) * Math.sin(angularRadius) * Math.cos(bearing)
    );
    const lon2 =
      lonRad +
      Math.atan2(
        Math.sin(bearing) * Math.sin(angularRadius) * Math.cos(latRad),
        Math.cos(angularRadius) - Math.sin(latRad) * Math.sin(lat2)
      );
    points.push({ lat: (lat2 * 180) / Math.PI, lon: (lon2 * 180) / Math.PI });
  }
  return points;
}
