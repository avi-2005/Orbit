export default function SatelliteInfoPanel({ satellite, onClose }) {
  if (!satellite) return null;

  const rows = [
    ["Name", satellite.name || "—"],
    ["Altitude", `${Math.round(satellite.altitudeKm)} km`],
    ["Position", `${satellite.lat.toFixed(2)}, ${satellite.lon.toFixed(2)}`],
  ];

  return (
    <div className="absolute top-6 right-6 w-72 bg-black/70 backdrop-blur-md border border-white/10 rounded-xl p-5 text-sm shadow-2xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="mono text-sky-300 text-base tracking-wide">Satellite</h2>
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
      <p className="text-white/30 text-[11px] mt-3 leading-relaxed">
        Position computed live from public orbital data (two-body + J2
        propagation), not fetched from a tracking API.
      </p>
    </div>
  );
}
