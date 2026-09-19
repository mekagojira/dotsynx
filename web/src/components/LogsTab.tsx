import React, { useState, useEffect, useRef } from "react";
import {
  Terminal,
  RefreshCw,
  Search,
  ArrowDownUp,
  ShieldAlert,
  AlertTriangle,
  CheckCircle2,
  Info,
  X,
} from "lucide-react";
import { LogEntry } from "../types";
import { fetchLogs } from "../api";

export const LogsTab: React.FC = () => {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterLevel, setFilterLevel] = useState<string>("ALL");
  const [searchQuery, setSearchQuery] = useState("");
  const [sortDesc, setSortDesc] = useState(true);

  const loadLogs = async () => {
    setLoading(true);
    try {
      const data = await fetchLogs(300);
      setLogs(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLogs();
    const interval = setInterval(loadLogs, 3000); // 3s polling for live stream
    return () => clearInterval(interval);
  }, []);

  let processed = logs.filter((entry) => {
    const matchesLevel = filterLevel === "ALL" || entry.level === filterLevel;
    const matchesSearch =
      !searchQuery ||
      entry.raw.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesLevel && matchesSearch;
  });

  if (!sortDesc) {
    processed = [...processed].reverse();
  }

  const getLevelBadge = (level: string) => {
    switch (level) {
      case "ERROR":
        return (
          <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-rose-500/20 text-rose-400 border border-rose-500/30">
            ERROR
          </span>
        );
      case "WARN":
        return (
          <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-500/20 text-amber-400 border border-amber-500/30">
            WARN
          </span>
        );
      case "SYNC":
        return (
          <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
            SYNC
          </span>
        );
      default:
        return (
          <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
            INFO
          </span>
        );
    }
  };

  return (
    <div className="space-y-4">
      {/* Controls Bar */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 sm:pb-0">
          {["ALL", "ERROR", "WARN", "SYNC", "INFO"].map((lvl) => (
            <button
              key={lvl}
              onClick={() => setFilterLevel(lvl)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                filterLevel === lvl
                  ? "bg-indigo-600 text-white"
                  : "bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700"
              }`}
            >
              {lvl}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-3">
          <div className="relative flex-1 sm:w-64">
            <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              placeholder="Search logs (e.g. error, push)..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-8 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>

          <button
            onClick={() => setSortDesc(!sortDesc)}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 text-xs text-slate-600 dark:text-slate-300 transition-colors"
            title="Toggle sort direction"
          >
            <ArrowDownUp className="w-3.5 h-3.5 text-indigo-500" />
            <span>{sortDesc ? "Newest First" : "Oldest First"}</span>
          </button>

          <button
            onClick={loadLogs}
            className="p-1.5 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 text-slate-500"
            title="Refresh logs"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      {/* Terminal View */}
      <div className="bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
        {/* Terminal top header */}
        <div className="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 flex items-center justify-between text-xs text-slate-400 font-mono">
          <div className="flex items-center gap-2">
            <div className="flex gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full bg-rose-500/80"></span>
              <span className="w-2.5 h-2.5 rounded-full bg-amber-500/80"></span>
              <span className="w-2.5 h-2.5 rounded-full bg-emerald-500/80"></span>
            </div>
            <span className="ml-2 font-semibold text-slate-300">
              ~/.local/share/dotsynx/dotsynx.log
            </span>
          </div>
          <span>{processed.length} line(s)</span>
        </div>

        {/* Output lines */}
        <div className="p-4 font-mono text-xs text-slate-300 max-h-[550px] overflow-y-auto space-y-1.5 leading-relaxed">
          {processed.length === 0 ? (
            <div className="py-12 text-center text-slate-600">
              No log entries match the current filter.
            </div>
          ) : (
            processed.map((entry, idx) => (
              <div
                key={idx}
                className="flex items-start gap-2.5 hover:bg-slate-900/50 px-1 py-0.5 rounded"
              >
                <span className="text-slate-500 shrink-0 select-none">
                  {new Date(entry.timestamp).toLocaleTimeString()}
                </span>
                <span className="shrink-0">{getLevelBadge(entry.level)}</span>
                <span
                  className={`break-all ${
                    entry.level === "ERROR"
                      ? "text-rose-300 font-semibold"
                      : entry.level === "WARN"
                        ? "text-amber-300"
                        : entry.level === "SYNC"
                          ? "text-emerald-300"
                          : "text-slate-300"
                  }`}
                >
                  {entry.message}
                </span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
