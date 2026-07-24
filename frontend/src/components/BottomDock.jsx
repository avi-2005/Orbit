import { useState } from "react";
import InsightsFeed from "./InsightsFeed";
import CopilotPanel from "./CopilotPanel";

export default function BottomDock({ insights, onSelectInsight }) {
  const [active, setActive] = useState(null); // null | "insights" | "copilot"

  const toggle = (tab) => setActive((a) => (a === tab ? null : tab));

  return (
    <div className="absolute bottom-3 left-1/2 -translate-x-1/2 w-[calc(100vw-1.5rem)] sm:w-auto sm:max-w-[480px] z-20">
      {active && (
        <div className="mb-2 bg-black/80 backdrop-blur-md border border-white/10 rounded-xl p-4">
          {active === "insights" && (
            <InsightsFeed insights={insights} onSelect={onSelectInsight} />
          )}
          {active === "copilot" && <CopilotPanel />}
        </div>
      )}

      <div className="flex bg-black/70 backdrop-blur-md border border-white/10 rounded-full p-1 mono text-xs">
        <button
          onClick={() => toggle("insights")}
          className={`flex-1 px-3 py-1.5 rounded-full transition-colors ${
            active === "insights" ? "bg-white/15 text-white" : "text-white/50 hover:text-white/80"
          }`}
        >
          Insights {insights.length > 0 && `(${insights.length})`}
        </button>
        <button
          onClick={() => toggle("copilot")}
          className={`flex-1 px-3 py-1.5 rounded-full transition-colors ${
            active === "copilot" ? "bg-white/15 text-white" : "text-white/50 hover:text-white/80"
          }`}
        >
          Copilot
        </button>
      </div>
    </div>
  );
}
