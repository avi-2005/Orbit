import { useEffect, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080";

const EXAMPLE_QUESTIONS = [
  "Any flights near a watch zone?",
  "Any flights at elevated risk right now?",
  "Any anomalies detected recently?",
];

export default function CopilotPanel() {
  const [question, setQuestion] = useState("");
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(false);
  const [lifetimeCount, setLifetimeCount] = useState(null);

  useEffect(() => {
    fetch(`${API_BASE}/api/stats`)
      .then((res) => res.json())
      .then((data) => setLifetimeCount(data.totalInsightsAllTime))
      .catch(() => {});
  }, []);

  const ask = async () => {
    const q = question.trim();
    if (!q || loading) return;

    setMessages((m) => [...m, { role: "user", text: q }]);
    setQuestion("");
    setLoading(true);

    try {
      const res = await fetch(`${API_BASE}/api/ask`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question: q }),
      });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || `Request failed (${res.status})`);
      }
      const data = await res.json();
      setMessages((m) => [...m, { role: "assistant", text: data.answer }]);
    } catch (err) {
      setMessages((m) => [
        ...m,
        { role: "assistant", text: `Copilot error: ${err.message}` },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="text-xs">
      <h3 className="mono text-white/50 tracking-widest mb-2 flex justify-between">
        <span>ORBIT COPILOT</span>
        {lifetimeCount !== null && (
          <span className="text-white/30 normal-case tracking-normal">
            {lifetimeCount.toLocaleString()} logged
          </span>
        )}
      </h3>

      {messages.length === 0 && (
        <div className="flex flex-wrap gap-1.5 mb-2">
          {EXAMPLE_QUESTIONS.map((q) => (
            <button
              key={q}
              onClick={() => setQuestion(q)}
              className="text-white/40 hover:text-white/80 border border-white/10 rounded-full px-2 py-0.5 transition-colors"
            >
              {q}
            </button>
          ))}
        </div>
      )}

      {messages.length > 0 && (
        <div className="max-h-[35vh] sm:max-h-32 overflow-y-auto space-y-1.5 mb-2">
          {messages.map((m, i) => (
            <p key={i} className={m.role === "user" ? "text-white/80" : "text-emerald-300"}>
              {m.role === "user" ? "› " : "» "}
              {m.text}
            </p>
          ))}
          {loading && <p className="text-white/30">thinking…</p>}
        </div>
      )}

      <div className="flex gap-2">
        <input
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && ask()}
          placeholder="Ask about current airspace…"
          className="flex-1 min-w-0 bg-white/5 border border-white/10 rounded px-2 py-1.5 mono text-white/90 outline-none focus:border-orange-400/50"
        />
        <button
          onClick={ask}
          disabled={loading}
          className="px-3 py-1.5 bg-orange-500/80 hover:bg-orange-500 disabled:opacity-50 rounded mono text-white transition-colors shrink-0"
        >
          Ask
        </button>
      </div>
    </div>
  );
}
