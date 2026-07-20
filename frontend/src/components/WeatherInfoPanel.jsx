export default function WeatherInfoPanel({ point, onClose }) {
  if (!point) return null;

  const kind =
    point.temperatureC >= 38 ? "Heatwave" : point.precipitationMm > 0.1 || point.windSpeedKmh > 35 ? "Storm activity" : "Calm";

  return (
    <div className="absolute top-6 right-6 w-72 bg-black/70 backdrop-blur-md border border-white/10 rounded-xl p-5 text-sm shadow-2xl">
      <div className="flex items-center justify-between mb-3">
        <h2 className="mono text-sky-300 text-base tracking-wide">{kind}</h2>
        <button onClick={onClose} className="text-white/40 hover:text-white transition-colors">
          ✕
        </button>
      </div>
      <dl className="space-y-1.5">
        <div className="flex justify-between gap-4">
          <dt className="text-white/40">Position</dt>
          <dd className="mono text-white/90">{point.lat.toFixed(1)}, {point.lon.toFixed(1)}</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-white/40">Temperature</dt>
          <dd className="mono text-white/90">{point.temperatureC.toFixed(1)} °C</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-white/40">Precipitation</dt>
          <dd className="mono text-white/90">{point.precipitationMm.toFixed(1)} mm</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-white/40">Wind speed</dt>
          <dd className="mono text-white/90">{point.windSpeedKmh.toFixed(0)} km/h</dd>
        </div>
      </dl>
    </div>
  );
}
