export default function ShipInfoPanel({ ship, onClose }) {
  if (!ship) return null;

  const rows = [
    ["Name", ship.name?.trim() || "—"],
    ["MMSI", ship.mmsi || "—"],
    ["Speed", `${ship.speedKn?.toFixed(1) ?? "—"} kn`],
    ["Course", `${Math.round(ship.courseDeg || 0)}°`],
    ["Position", `${ship.lat.toFixed(2)}, ${ship.lon.toFixed(2)}`],
  ];

  return (
    <div className="absolute top-6 right-6 w-72 bg-black/70 backdrop-blur-md border border-white/10 rounded-xl p-5 text-sm shadow-2xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="mono text-emerald-300 text-base tracking-wide">Vessel</h2>
        <button onClick={onClose} className="text-white/40 hover:text-white transition-colors">
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
        Live AIS position report via aisstream.io.
      </p>
    </div>
  );
}
