const ITEMS = [
  { swatch: "bg-orange-400", shape: "✈ plane", label: "Flight — aircraft silhouette, points along real heading" },
  { swatch: "bg-emerald-400", shape: "⛴ boat", label: "Ship — vessel silhouette, points along AIS course" },
  { swatch: "bg-sky-300", shape: "◆ satellite", label: "Satellite — body + solar panels, real orbital position" },
  { swatch: "bg-orange-500", shape: "● pulsing", label: "Heatwave — orange, brighter = hotter" },
  { swatch: "bg-sky-400/50", shape: "● pulsing", label: "Storm — blue, click for exact rain/wind" },
  { swatch: "bg-yellow-300", shape: "○ circle", label: "Watch zone / trade chokepoint" },
  { swatch: "bg-red-500", shape: "◎ pulsing ring", label: "Elevated risk — combines zone + weather + anomaly signals" },
];

export default function LegendPanel({ open, onToggle }) {
  return (
    <div className="absolute top-6 right-6">
      <button
        onClick={onToggle}
        className="w-7 h-7 rounded-full bg-black/70 border border-white/10 text-white/60 hover:text-white mono text-xs flex items-center justify-center transition-colors"
      >
        i
      </button>

      {open && (
        <div className="mt-2 w-80 bg-black/80 backdrop-blur-md border border-white/10 rounded-xl p-4 text-xs">
          <h3 className="mono text-white/50 tracking-widest mb-2">LEGEND</h3>
          <ul className="space-y-1.5 mb-4">
            {ITEMS.map((item) => (
              <li key={item.label} className="flex items-center gap-2 text-white/70">
                <span className={`w-2.5 h-2.5 rounded-full ${item.swatch}`} />
                {item.label}
              </li>
            ))}
          </ul>

          <h3 className="mono text-white/50 tracking-widest mb-1.5">WHY ORBIT</h3>
          <p className="text-white/60 leading-relaxed">
            A live situational-awareness view of global movement — flights,
            ships, satellites, and weather in one place — with automatic
            anomaly detection, trade-chokepoint congestion tracking (Hormuz,
            Suez, Malacca), and an AI copilot that reasons over what's
            actually happening right now.
          </p>
        </div>
      )}
    </div>
  );
}
