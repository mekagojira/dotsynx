import React, { useState } from "react";
import {
  Sparkles,
  Folder,
  FileText,
  Plus,
  ShieldAlert,
  Check,
  Search,
  Filter,
} from "lucide-react";
import { Suggestion } from "../types";

interface SuggestionsTabProps {
  suggestions: Suggestion[];
  trackedPaths: Set<string>;
  onTrack: (path: string, isDir: boolean, desc: string) => void;
}

export const SuggestionsTab: React.FC<SuggestionsTabProps> = ({
  suggestions = [],
  trackedPaths,
  onTrack,
}) => {
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [searchTerm, setSearchTerm] = useState("");

  const safeSuggestions = suggestions || [];
  const categories = [
    "all",
    ...Array.from(
      new Set(safeSuggestions.map((s) => s.category).filter(Boolean)),
    ),
  ];

  const filtered = safeSuggestions.filter((s) => {
    const matchesCat =
      selectedCategory === "all" || s.category === selectedCategory;
    const matchesSearch =
      (s?.path || "").toLowerCase().includes(searchTerm.toLowerCase()) ||
      (s?.description || "").toLowerCase().includes(searchTerm.toLowerCase());
    return matchesCat && matchesSearch;
  });

  return (
    <div className="space-y-4">
      {/* Banner */}
      <div className="bg-gradient-to-r from-indigo-500/10 via-purple-500/10 to-transparent p-5 rounded-2xl border border-indigo-100 dark:border-indigo-950 flex items-start gap-4">
        <div className="p-2.5 rounded-xl bg-indigo-600 text-white shadow-md shadow-indigo-600/20 shrink-0">
          <Sparkles className="w-5 h-5" />
        </div>
        <div>
          <h3 className="text-base font-bold text-slate-900 dark:text-white">
            Auto-Discovered Dotfiles on This Computer
          </h3>
          <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
            dotsynx inspected your system for standard shells, editors,
            terminals, and configs. Add them to your sync repository with one
            click.
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3 items-stretch sm:items-center justify-between bg-white dark:bg-slate-900 p-4 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 sm:pb-0">
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setSelectedCategory(cat)}
              className={`text-xs px-3 py-1.5 rounded-lg font-medium capitalize shrink-0 transition-colors ${
                selectedCategory === cat
                  ? "bg-indigo-600 text-white"
                  : "bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300"
              }`}
            >
              {cat}
            </button>
          ))}
        </div>

        <div className="relative w-full sm:w-64">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            type="text"
            placeholder="Search suggestions..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-9 pr-3 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
          />
        </div>
      </div>

      {/* Suggestions Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filtered.map((s) => {
          const isTracked = trackedPaths.has(s.path);

          return (
            <div
              key={s.path}
              className={`bg-white dark:bg-slate-900 border rounded-xl p-4 flex flex-col justify-between shadow-sm transition-all ${
                isTracked
                  ? "border-slate-200 dark:border-slate-800 opacity-75"
                  : "border-slate-200 hover:border-indigo-300 dark:border-slate-800 dark:hover:border-indigo-700"
              }`}
            >
              <div>
                <div className="flex items-start justify-between gap-2 mb-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <div className="p-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 shrink-0">
                      {s.is_dir ? (
                        <Folder className="w-4 h-4 text-sky-500" />
                      ) : (
                        <FileText className="w-4 h-4 text-indigo-500" />
                      )}
                    </div>
                    <span className="font-mono text-sm font-semibold text-slate-900 dark:text-white truncate">
                      ~/{s.path}
                    </span>
                  </div>
                  <span className="text-[10px] px-2 py-0.5 rounded font-medium bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 shrink-0">
                    {s.category}
                  </span>
                </div>

                <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2">
                  {s.description}
                </p>

                {s.is_sensitive && (
                  <div className="mt-2 flex items-center gap-1 text-[11px] text-amber-600 dark:text-amber-400 font-medium">
                    <ShieldAlert className="w-3.5 h-3.5 shrink-0" />
                    <span>May contain sensitive credentials</span>
                  </div>
                )}
              </div>

              <div className="mt-4 pt-3 border-t border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                <span className="text-[11px] text-slate-400">
                  {s.is_dir ? "Directory" : "File"}
                </span>

                {isTracked ? (
                  <span className="flex items-center gap-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
                    <Check className="w-3.5 h-3.5" /> Tracked
                  </span>
                ) : (
                  <button
                    onClick={() => onTrack(s.path, s.is_dir, s.description)}
                    className="flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-medium bg-indigo-50 hover:bg-indigo-100 text-indigo-700 dark:bg-indigo-950/60 dark:hover:bg-indigo-900/80 dark:text-indigo-300 transition-colors"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>Track</span>
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
