/**
 * Returns the latitude/longitude directly under the sun right now (the
 * "subsolar point"). Standard approximate astronomical formulas — accurate
 * to within about a degree, which is plenty for a visual day/night line.
 */
export function getSubsolarPoint(date = new Date()) {
  const rad = Math.PI / 180;

  const startOfYear = Date.UTC(date.getUTCFullYear(), 0, 1);
  const dayOfYear = (date.getTime() - startOfYear) / 86400000;

  // Approximate solar declination (how far north/south the sun sits).
  const declination = -23.44 * Math.cos(rad * (360 / 365) * (dayOfYear + 10));

  const utcHours =
    date.getUTCHours() + date.getUTCMinutes() / 60 + date.getUTCSeconds() / 3600;

  // The sun sits over longitude 0 at UTC solar noon, and sweeps 15°/hour.
  let lon = (12 - utcHours) * 15;
  lon = ((lon + 180) % 360 + 360) % 360 - 180; // normalize to [-180, 180]

  return { lat: declination, lon };
}
