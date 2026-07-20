import { projectPosition } from "../utils/geo";

export default function InfoPanel({ flight, onClose }) {
  if (!flight) return null;

  const projected =
    flight.velocity && flight.heading !== undefined
      ? projectPosition(flight.lat, flight.lon, flight.heading || 0, flight.velocity || 0, 600, 1)
      : null;

  const rows = [
    ["Callsign", flight.callsign?.trim() || "—"],
    ["ICAO24", flight.icao24 || "—"],
    ["Origin country", flight.originCountry || "—"],
    ["Altitude", flight.altitude ? `${Math.round(flight.altitude)} m` : "—"],
    ["Speed", flight.velocity ? `${Math.round(flight.velocity * 3.6)} km/h` : "—"],
    ["Heading", flight.heading ? `${Math.round(flight.heading)}°` : "—"],
    ["Position", `${flight.lat.toFixed(2)}, ${flight.lon.toFixed(2)}`],
    ["On ground", flight.onGround ? "Yes" : "No"],
    ...(projected
      ? [["Projected (+10 min)", `${projected.lat.toFixed(2)}, ${projected.lon.toFixed(2)}`]]
      : []),
  ];

  const risk = flight.riskScore || 0;
  const riskColor = risk >= 50 ? "text-red-400" : risk >= 20 ? "text-amber-400" : "text-white/40";

  return (
    <div className="absolute top-6 right-6 w-72 bg-black/70 backdrop-blur-md border border-white/10 rounded-xl p-5 text-sm shadow-2xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="mono text-orange-400 text-base tracking-wide">
          {flight.callsign?.trim() || "UNKNOWN"}
        </h2>
        <button
          onClick={onClose}
          className="text-white/40 hover:text-white transition-colors"
        >
          ✕
        </button>
      </div>
      <dl className="space-y-1.5">
        {rows.map(([label, value]) => (
          <div key={label} className="flex justify-between gap-4">
            <dt className="text-white/40">{label}</dt>
            <dd className="mono text-white/90 text-right">{value}</dd>
          </div>
        ))}
      </dl>

      {risk > 0 && (
        <div className="mt-3 pt-3 border-t border-white/10">
          <div className="flex justify-between">
            <span className="text-white/40">Risk score</span>
            <span className={`mono ${riskColor}`}>{risk}/100</span>
          </div>
          {flight.riskFactors?.map((f) => (
            <p key={f} className="text-white/40 text-[11px] mt-1">• {f}</p>
          ))}
        </div>
      )}
    </div>
  );
}
