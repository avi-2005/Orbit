const LAYERS = [
  { key: "flights", label: "Flights" },
  { key: "ships", label: "Ships" },
  { key: "satellites", label: "Satellites" },
  { key: "weather", label: "Weather" },
];

export default function FilterPanel({ visible, onToggleLayer, countryFilter, onCountryFilterChange }) {
  return (
    <div className="absolute top-6 left-1/2 -translate-x-1/2 flex items-center gap-3 bg-black/70 backdrop-blur-md border border-white/10 rounded-full px-4 py-2 text-xs">
      {LAYERS.map((l) => (
        <button
          key={l.key}
          onClick={() => onToggleLayer(l.key)}
          className={`mono px-2 py-1 rounded-full transition-colors ${
            visible[l.key] ? "bg-white/15 text-white" : "text-white/30 hover:text-white/60"
          }`}
        >
          {l.label}
        </button>
      ))}
      <span className="text-white/20">|</span>
      <input
        value={countryFilter}
        onChange={(e) => onCountryFilterChange(e.target.value)}
        placeholder="Filter by country…"
        className="bg-transparent border-b border-white/20 focus:border-white/50 outline-none text-white/80 placeholder:text-white/30 w-32 mono"
      />
    </div>
  );
}
