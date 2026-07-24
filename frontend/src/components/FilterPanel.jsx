const LAYERS = [
  { key: "flights", label: "Flights" },
  { key: "satellites", label: "Satellites" },
  { key: "weather", label: "Weather" },
];

export default function FilterPanel({ visible, onToggleLayer }) {
  return (
    <div className="absolute top-[4.5rem] left-1/2 -translate-x-1/2 sm:top-6 flex flex-wrap justify-center items-center gap-1.5 sm:gap-3 bg-black/70 backdrop-blur-md border border-white/10 rounded-2xl sm:rounded-full px-2.5 sm:px-4 py-1.5 sm:py-2 text-[10px] sm:text-xs max-w-[92vw]">
      {LAYERS.map((l) => (
        <button
          key={l.key}
          onClick={() => onToggleLayer(l.key)}
          className={`mono px-1.5 sm:px-2 py-0.5 sm:py-1 rounded-full whitespace-nowrap transition-colors ${
            visible[l.key] ? "bg-white/15 text-white" : "text-white/30 hover:text-white/60"
          }`}
        >
          {l.label}
        </button>
      ))}
    </div>
  );
}
