const ITEMS = [
  { swatch: "bg-orange-400", shape: "✈ plane", label: "Flight — aircraft silhouette, points along real heading" },
  { swatch: "bg-sky-300", shape: "◆ satellite", label: "Satellite — body + solar panels, real orbital position" },
  { swatch: "bg-orange-500", shape: "● pulsing", label: "Heatwave — orange, brighter = hotter" },
  { swatch: "bg-sky-400/50", shape: "● pulsing", label: "Storm — blue, click for exact rain/wind" },
  { swatch: "bg-yellow-300", shape: "○ circle", label: "Watch zone" },
  { swatch: "bg-red-500", shape: "◎ pulsing ring", label: "Elevated risk — combines zone + weather + anomaly signals" },
];

export default function LegendPanel({ open, onToggle }) {
  return (
    <div className="absolute top-3 right-3 sm:top-6 sm:right-6 z-20">
      <button
        onClick={onToggle}
        className="w-6 h-6 sm:w-7 sm:h-7 rounded-full bg-black/70 border border-white/10 text-white/60 hover:text-white mono text-xs flex items-center justify-center transition-colors"
      >
        i
      </button>

      {open && (
        <div className="mt-2 w-[calc(100vw-1.5rem)] max-w-80 bg-black/80 backdrop-blur-md border border-white/10 rounded-xl p-4 text-xs">
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
            A live situational-awareness view of global airspace — flights,
            satellites, and weather in one place — with automatic anomaly
            detection and an AI copilot that reasons over what's actually
            happening right now.
          </p>
        </div>
      )}
    </div>
  );
}
