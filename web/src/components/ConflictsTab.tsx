import React from "react";
import {
  AlertTriangle,
  CheckCircle2,
  ArrowRight,
  ShieldCheck,
} from "lucide-react";
import { ConflictItem } from "../types";

interface ConflictsTabProps {
  conflicts: ConflictItem[];
  onResolve: (path: string, strategy: "ours" | "theirs") => void;
}

export const ConflictsTab: React.FC<ConflictsTabProps> = ({
  conflicts = [],
  onResolve,
}) => {
  const safeConflicts = conflicts || [];
  if (safeConflicts.length === 0) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-12 text-center shadow-sm">
        <div className="w-12 h-12 rounded-full bg-emerald-50 dark:bg-emerald-950/60 text-emerald-600 dark:text-emerald-400 mx-auto flex items-center justify-center mb-3">
          <ShieldCheck className="w-6 h-6" />
        </div>
        <h3 className="text-lg font-bold text-slate-900 dark:text-white">
          No Divergence or Conflicts!
        </h3>
        <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1">
          Your local dotfiles and remote Git repository are completely
          synchronized.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/50 p-4 rounded-xl flex items-start gap-3">
        <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
        <div>
          <h4 className="text-sm font-semibold text-amber-800 dark:text-amber-300">
            {safeConflicts.length} Conflict(s) Detected
          </h4>
          <p className="text-xs text-amber-700 dark:text-amber-400 mt-0.5">
            Changes on this machine diverge from the remote Git repository.
            Choose which version to keep. Before any action is taken, dotsynx
            automatically creates a local safety snapshot.
          </p>
        </div>
      </div>

      <div className="space-y-4">
        {safeConflicts.map((c) => (
          <div
            key={c.path}
            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-sm"
          >
            {/* Conflict header */}
            <div className="p-4 border-b border-slate-100 dark:border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-slate-50/50 dark:bg-slate-800/30">
              <div>
                <span className="font-mono text-sm font-bold text-slate-900 dark:text-white">
                  {c.path}
                </span>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Local: {c.home_path}
                </p>
              </div>

              {/* Resolution options */}
              <div className="flex items-center gap-2">
                <button
                  onClick={() => onResolve(c.path, "ours")}
                  className="px-3 py-1.5 rounded-lg text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white transition-colors shadow-sm"
                >
                  Keep Local (Ours)
                </button>
                <button
                  onClick={() => onResolve(c.path, "theirs")}
                  className="px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-200 hover:bg-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 transition-colors"
                >
                  Use Remote (Theirs)
                </button>
              </div>
            </div>

            {/* Diff Viewer */}
            <div className="p-4 bg-slate-950 font-mono text-xs text-slate-200 overflow-x-auto max-h-96">
              <pre className="whitespace-pre">
                {c.diff.split("\n").map((line, idx) => {
                  let color = "text-slate-400";
                  if (line.startsWith("+"))
                    color = "text-emerald-400 bg-emerald-950/40";
                  if (line.startsWith("-"))
                    color = "text-rose-400 bg-rose-950/40";
                  if (line.startsWith("@")) color = "text-indigo-400";
                  return (
                    <div key={idx} className={`${color} px-1 rounded`}>
                      {line || " "}
                    </div>
                  );
                })}
              </pre>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
