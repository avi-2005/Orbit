import { useMemo, useState } from "react";
import { Canvas } from "@react-three/fiber";
import { OrbitControls, Stars } from "@react-three/drei";
import Globe from "./components/Globe";
import FlightPoints from "./components/FlightPoints";
import ShipsLayer from "./components/ShipsLayer";
import ZonesLayer from "./components/ZonesLayer";
import SatellitesLayer from "./components/SatellitesLayer";
import WeatherLayer from "./components/WeatherLayer";
import InfoPanel from "./components/InfoPanel";
import SatelliteInfoPanel from "./components/SatelliteInfoPanel";
import ShipInfoPanel from "./components/ShipInfoPanel";
import WeatherInfoPanel from "./components/WeatherInfoPanel";
import LegendPanel from "./components/LegendPanel";
import FilterPanel from "./components/FilterPanel";
import InsightsFeed from "./components/InsightsFeed";
import CopilotPanel from "./components/CopilotPanel";
import { useFlightSocket } from "./hooks/useFlightSocket";

export default function App() {
  const { flights, insights, satellites, weather, ships, status } = useFlightSocket();
  const [selectedFlight, setSelectedFlight] = useState(null);
  const [selectedSatellite, setSelectedSatellite] = useState(null);
  const [selectedWeather, setSelectedWeather] = useState(null);
  const [selectedShip, setSelectedShip] = useState(null);
  const [legendOpen, setLegendOpen] = useState(false);
  const [visible, setVisible] = useState({ flights: true, ships: true, satellites: true, weather: true });
  const [countryFilter, setCountryFilter] = useState("");

  const clearSelections = () => {
    setSelectedFlight(null);
    setSelectedSatellite(null);
    setSelectedWeather(null);
    setSelectedShip(null);
  };
  const selectFlight = (f) => { clearSelections(); setSelectedFlight(f); };
  const selectSatellite = (s) => { clearSelections(); setSelectedSatellite(s); };
  const selectWeather = (w) => { clearSelections(); setSelectedWeather(w); };
  const selectShip = (s) => { clearSelections(); setSelectedShip(s); };

  const toggleLayer = (key) => setVisible((v) => ({ ...v, [key]: !v[key] }));

  const filteredFlights = useMemo(() => {
    if (!countryFilter.trim()) return flights;
    const q = countryFilter.trim().toLowerCase();
    return flights.filter((f) => f.originCountry?.toLowerCase().includes(q));
  }, [flights, countryFilter]);

  return (
    <div className="relative w-screen h-screen bg-[#05060a]">
      <Canvas camera={{ position: [0, 0, 5.5], fov: 45 }}>
        <ambientLight intensity={0.6} />
        <directionalLight position={[5, 3, 5]} intensity={1.2} />
        <Stars radius={80} depth={40} count={3000} factor={2} fade speed={0.5} />
        <Globe />
        <ZonesLayer />
        {visible.weather && <WeatherLayer weather={weather} onSelect={selectWeather} />}
        {visible.flights && (
          <FlightPoints flights={filteredFlights} selectedIcao={selectedFlight?.icao24} onSelect={selectFlight} />
        )}
        {visible.ships && <ShipsLayer ships={ships} onSelect={selectShip} />}
        {visible.satellites && <SatellitesLayer satellites={satellites} onSelect={selectSatellite} />}
        <OrbitControls enablePan={false} minDistance={2.2} maxDistance={10} rotateSpeed={0.4} />
      </Canvas>

      {/* Header */}
      <div className="absolute top-6 left-6">
        <h1 className="mono text-2xl tracking-[0.3em] text-white/90">ORBIT</h1>
        <p className="text-white/40 text-xs mt-1">live global movement intelligence</p>
      </div>

      <FilterPanel
        visible={visible}
        onToggleLayer={toggleLayer}
        countryFilter={countryFilter}
        onCountryFilterChange={setCountryFilter}
      />

      {/* Status bar */}
      <div className="absolute bottom-6 left-6 flex items-center gap-3 mono text-xs text-white/50">
        <span
          className={`inline-block w-2 h-2 rounded-full ${
            status === "connected" ? "bg-emerald-400" : "bg-amber-400 animate-pulse"
          }`}
        />
        {status === "connected" ? "live" : status}
        <span className="text-white/20">|</span>
        {filteredFlights.length.toLocaleString()} aircraft
        <span className="text-white/20">|</span>
        {ships.length.toLocaleString()} ships
        <span className="text-white/20">|</span>
        {satellites.length.toLocaleString()} satellites
      </div>

      <LegendPanel open={legendOpen} onToggle={() => setLegendOpen((o) => !o)} />
      {!legendOpen && <InfoPanel flight={selectedFlight} onClose={() => setSelectedFlight(null)} />}
      {!legendOpen && (
        <SatelliteInfoPanel satellite={selectedSatellite} onClose={() => setSelectedSatellite(null)} />
      )}
      {!legendOpen && <ShipInfoPanel ship={selectedShip} onClose={() => setSelectedShip(null)} />}
      {!legendOpen && (
        <WeatherInfoPanel point={selectedWeather} onClose={() => setSelectedWeather(null)} />
      )}
      <InsightsFeed insights={insights} onSelect={selectFlight} />
      <CopilotPanel />
    </div>
  );
}
