import React from "react";
import {
  RefreshCw,
  Moon,
  Sun,
  Monitor,
  Play,
  CheckCircle2,
  AlertCircle,
} from "lucide-react";
import { useTheme } from "./ThemeContext";
import { SystemStatus } from "../types";

interface HeaderProps {
  status: SystemStatus | null;
  syncing: boolean;
  onSync: () => void;
  lastMessage?: string;
  isError?: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  status,
  syncing,
  onSync,
  lastMessage,
  isError,
}) => {
  const { theme, setTheme, isDark } = useTheme();

  return (
    <header className="border-b border-slate-200 dark:border-slate-800 bg-white/75 dark:bg-slate-900/75 backdrop-blur sticky top-0 z-20">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo & title */}
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-indigo-400 flex items-center justify-center text-white font-bold shadow-md shadow-indigo-500/20">
              ⚡
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-white">
                  dotsynx
                </h1>
                <span className="text-xs px-2 py-0.5 rounded-full font-medium bg-indigo-50 text-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800">
                  v1.0
                </span>
              </div>
            </div>
          </div>

          {/* Center notification banner if present */}
          {lastMessage && (
            <div
              className={`hidden md:flex items-center gap-2 text-xs px-3 py-1.5 rounded-lg max-w-md truncate border ${
                isError
                  ? "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-950/50 dark:text-rose-300 dark:border-rose-900"
                  : "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-900"
              }`}
            >
              {isError ? (
                <AlertCircle className="w-3.5 h-3.5 shrink-0" />
              ) : (
                <CheckCircle2 className="w-3.5 h-3.5 shrink-0" />
              )}
              <span className="truncate">{lastMessage}</span>
            </div>
          )}

          {/* Right actions: Sync button & Theme selector */}
          <div className="flex items-center gap-3">
            <button
              onClick={onSync}
              disabled={syncing}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-sm font-medium shadow-sm transition-all ${
                syncing
                  ? "bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-500 cursor-not-allowed"
                  : "bg-indigo-600 hover:bg-indigo-500 text-white shadow-indigo-600/20 hover:shadow active:scale-95"
              }`}
            >
              <RefreshCw
                className={`w-4 h-4 ${syncing ? "animate-spin" : ""}`}
              />
              <span>{syncing ? "Syncing..." : "Sync Now"}</span>
            </button>

            {/* Theme switcher */}
            <div className="flex items-center bg-slate-100 dark:bg-slate-800 p-1 rounded-lg border border-slate-200 dark:border-slate-700">
              <button
                title="System theme"
                onClick={() => setTheme("system")}
                className={`p-1.5 rounded-md transition-colors ${
                  theme === "system"
                    ? "bg-white dark:bg-slate-700 text-indigo-600 dark:text-indigo-400 shadow-sm"
                    : "text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                }`}
              >
                <Monitor className="w-4 h-4" />
              </button>
              <button
                title="Light mode"
                onClick={() => setTheme("light")}
                className={`p-1.5 rounded-md transition-colors ${
                  theme === "light"
                    ? "bg-white text-indigo-600 shadow-sm"
                    : "text-slate-400 hover:text-slate-600"
                }`}
              >
                <Sun className="w-4 h-4" />
              </button>
              <button
                title="Dark mode"
                onClick={() => setTheme("dark")}
                className={`p-1.5 rounded-md transition-colors ${
                  theme === "dark"
                    ? "bg-slate-700 text-indigo-400 shadow-sm"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                <Moon className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
};
