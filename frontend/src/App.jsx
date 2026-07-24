import { useState, Suspense } from "react";
import { Canvas } from "@react-three/fiber";
import { OrbitControls, Stars } from "@react-three/drei";
import Globe from "./components/Globe";
import FlightPoints from "./components/FlightPoints";
import ZonesLayer from "./components/ZonesLayer";
import SatellitesLayer from "./components/SatellitesLayer";
import WeatherLayer from "./components/WeatherLayer";
import GlobeErrorBoundary from "./components/GlobeErrorBoundary";
import InfoPanel from "./components/InfoPanel";
import SatelliteInfoPanel from "./components/SatelliteInfoPanel";
import WeatherInfoPanel from "./components/WeatherInfoPanel";
import LegendPanel from "./components/LegendPanel";
import FilterPanel from "./components/FilterPanel";
import BottomDock from "./components/BottomDock";
import { useFlightSocket } from "./hooks/useFlightSocket";

export default function App() {
  const { flights, insights, satellites, weather, status } = useFlightSocket();
  const [selectedFlight, setSelectedFlight] = useState(null);
  const [selectedSatellite, setSelectedSatellite] = useState(null);
  const [selectedWeather, setSelectedWeather] = useState(null);
  const [legendOpen, setLegendOpen] = useState(false);
  const [visible, setVisible] = useState({ flights: true, satellites: true, weather: true });

  const clearSelections = () => {
    setSelectedFlight(null);
    setSelectedSatellite(null);
    setSelectedWeather(null);
  };
  const selectFlight = (f) => { clearSelections(); setSelectedFlight(f); };
  const selectSatellite = (s) => { clearSelections(); setSelectedSatellite(s); };
  const selectWeather = (w) => { clearSelections(); setSelectedWeather(w); };

  const toggleLayer = (key) => setVisible((v) => ({ ...v, [key]: !v[key] }));

  return (
    <div className="relative w-screen h-screen bg-[#05060a]">
      <GlobeErrorBoundary>
        <Canvas camera={{ position: [2.245, 1.424, -4.815], fov: 45 }}>
          <ambientLight intensity={0.6} />
          <directionalLight position={[5, 3, 5]} intensity={1.2} />
          <Suspense fallback={null}>
            <Stars radius={80} depth={40} count={3000} factor={2} fade speed={0.5} />
            <Globe />
            <ZonesLayer />
            {visible.weather && <WeatherLayer weather={weather} onSelect={selectWeather} />}
            {visible.flights && (
              <FlightPoints flights={flights} selectedIcao={selectedFlight?.icao24} onSelect={selectFlight} />
            )}
            {visible.satellites && <SatellitesLayer satellites={satellites} onSelect={selectSatellite} />}
          </Suspense>
          <OrbitControls enablePan={false} minDistance={2.2} maxDistance={10} rotateSpeed={0.4} />
        </Canvas>
      </GlobeErrorBoundary>

      {/* Header */}
      <div className="absolute top-3 left-3 sm:top-6 sm:left-6">
        <h1 className="mono text-lg sm:text-2xl tracking-[0.2em] sm:tracking-[0.3em] text-white/90">ORBIT</h1>
        <p className="text-white/40 text-[10px] sm:text-xs mt-0.5 sm:mt-1 hidden sm:block">live global movement intelligence</p>
      </div>

      <FilterPanel visible={visible} onToggleLayer={toggleLayer} />

      {/* Status bar */}
      <div className="absolute bottom-[4.5rem] sm:bottom-6 left-3 sm:left-6 flex items-center gap-2 sm:gap-3 mono text-[10px] sm:text-xs text-white/50">
        <span
          className={`inline-block w-2 h-2 rounded-full ${
            status === "connected" ? "bg-emerald-400" : "bg-amber-400 animate-pulse"
          }`}
        />
        {status === "connected" ? "live" : status}
        <span className="text-white/20 hidden sm:inline">|</span>
        <span className="hidden sm:inline">{flights.length.toLocaleString()} aircraft</span>
        <span className="text-white/20 hidden sm:inline">|</span>
        <span className="hidden sm:inline">{satellites.length.toLocaleString()} satellites</span>
        <span className="sm:hidden">
          {flights.length.toLocaleString()}✈ {satellites.length.toLocaleString()}◆
        </span>
      </div>

      <LegendPanel open={legendOpen} onToggle={() => setLegendOpen((o) => !o)} />
      {!legendOpen && <InfoPanel flight={selectedFlight} onClose={() => setSelectedFlight(null)} />}
      {!legendOpen && (
        <SatelliteInfoPanel satellite={selectedSatellite} onClose={() => setSelectedSatellite(null)} />
      )}
      {!legendOpen && (
        <WeatherInfoPanel point={selectedWeather} onClose={() => setSelectedWeather(null)} />
      )}
      <BottomDock insights={insights} onSelectInsight={selectFlight} />
    </div>
  );
}
