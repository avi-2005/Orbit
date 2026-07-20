const SEVERITY_COLOR = {
  info: "text-sky-400",
  warning: "text-amber-400",
  alert: "text-red-400",
};

const TYPE_LABEL = {
  holding: "Holding pattern",
  descent: "Rapid descent",
  zone: "Zone entry",
};

export default function InsightsFeed({ insights, onSelect }) {
  return (
    <div className="absolute bottom-6 right-6 w-80 max-h-72 overflow-y-auto bg-black/70 backdrop-blur-md border border-white/10 rounded-xl p-4 text-xs">
      <h3 className="mono text-white/50 tracking-widest mb-2">LIVE INSIGHTS</h3>
      {insights.length === 0 && (
        <p className="text-white/30">Watching for anomalies…</p>
      )}
      <div className="space-y-2">
        {insights.map((ins) => (
          <button
            key={ins.id}
            onClick={() => onSelect(ins)}
            className="w-full text-left border-b border-white/5 pb-2 hover:bg-white/5 rounded px-1 -mx-1 transition-colors"
          >
            <div className="flex justify-between items-center">
              <span className={`mono ${SEVERITY_COLOR[ins.severity] || "text-white/60"}`}>
                {TYPE_LABEL[ins.type] || ins.type}
              </span>
              <span className="text-white/30">
                {ins.callsign?.trim() || ins.icao24}
              </span>
            </div>
            <p className="text-white/50 mt-0.5">{ins.detail}</p>
          </button>
        ))}
      </div>
    </div>
  );
}
