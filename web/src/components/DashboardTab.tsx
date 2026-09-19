import React from "react";
import {
  GitBranch,
  FolderGit2,
  Cpu,
  HardDrive,
  Clock,
  CheckCircle,
  AlertTriangle,
  ShieldCheck,
  ArrowUpRight,
  ArrowDownLeft,
  AlertCircle,
  CheckCircle2,
  ListFilter,
} from "lucide-react";
import { SystemStatus, SyncMode, SyncResult } from "../types";

interface DashboardTabProps {
  status: SystemStatus | null;
  lastSyncResult: SyncResult | null;
  onUpdateMode: (mode: SyncMode) => void;
  onSwitchTab: (tab: string) => void;
}

export const DashboardTab: React.FC<DashboardTabProps> = ({
  status,
  lastSyncResult,
  onUpdateMode,
  onSwitchTab,
}) => {
  if (!status) {
    return (
      <div className="py-12 text-center text-slate-400">
        Loading system status...
      </div>
    );
  }

  const { git, daemon, service, conflicts, config } = status;
  const isGitConfigured = Boolean(config.repo_url);
  const trackedCount = config.tracked ? config.tracked.length : 0;
  const conflictCount = conflicts ? conflicts.length : 0;

  return (
    <div className="space-y-6">
      {/* Top Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Sync Mode Card */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">
              Sync Mode
            </span>
            <Clock className="w-4 h-4 text-indigo-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold capitalize text-slate-900 dark:text-white">
              {config.sync_mode}
            </span>
            {config.sync_mode === "auto" && (
              <span className="text-xs text-slate-500 dark:text-slate-400">
                ({config.interval})
              </span>
            )}
          </div>
          <div className="mt-3 flex gap-1.5">
            {(["manual", "auto", "boot"] as SyncMode[]).map((m) => (
              <button
                key={m}
                onClick={() => onUpdateMode(m)}
                className={`text-xs px-2.5 py-1 rounded-md font-medium capitalize transition-colors ${
                  config.sync_mode === m
                    ? "bg-indigo-600 text-white"
                    : "bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300"
                }`}
              >
                {m}
              </button>
            ))}
          </div>
        </div>

        {/* Git Repository Card */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm">
          <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">
              Git Status
            </span>
            <GitBranch className="w-4 h-4 text-emerald-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold text-slate-900 dark:text-white">
              {git?.Branch || config.branch || "main"}
            </span>
            {git?.RemoteURL ? (
              <span className="text-xs text-emerald-600 dark:text-emerald-400 font-medium">
                Connected
              </span>
            ) : (
              <span className="text-xs text-amber-600 dark:text-amber-400 font-medium">
                Local Only
              </span>
            )}
          </div>
          <div className="mt-3 flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400">
            <span className="flex items-center gap-1">
              <ArrowUpRight className="w-3.5 h-3.5 text-indigo-500" /> Ahead:{" "}
              {git?.Ahead ?? 0}
            </span>
            <span className="flex items-center gap-1">
              <ArrowDownLeft className="w-3.5 h-3.5 text-indigo-500" /> Behind:{" "}
              {git?.Behind ?? 0}
            </span>
          </div>
        </div>

        {/* Tracked Files Card */}
        <div
          onClick={() => onSwitchTab("files")}
          className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm cursor-pointer hover:border-indigo-400 dark:hover:border-indigo-600 transition-colors"
        >
          <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">
              Tracked Dotfiles
            </span>
            <HardDrive className="w-4 h-4 text-sky-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold text-slate-900 dark:text-white">
              {trackedCount}
            </span>
            <span className="text-xs text-slate-500">items</span>
          </div>
          <p className="mt-3 text-xs text-indigo-600 dark:text-indigo-400 font-medium">
            Manage tracked files →
          </p>
        </div>

        {/* Conflicts / Health Card */}
        <div
          onClick={() =>
            onSwitchTab(conflictCount > 0 ? "conflicts" : "suggestions")
          }
          className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm cursor-pointer hover:border-indigo-400 dark:hover:border-indigo-600 transition-colors"
        >
          <div className="flex items-center justify-between text-slate-500 dark:text-slate-400 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">
              Sync Health
            </span>
            {conflictCount > 0 ? (
              <AlertTriangle className="w-4 h-4 text-amber-500" />
            ) : (
              <ShieldCheck className="w-4 h-4 text-emerald-500" />
            )}
          </div>
          <div className="flex items-baseline gap-2">
            <span
              className={`text-2xl font-bold ${conflictCount > 0 ? "text-amber-600" : "text-slate-900 dark:text-white"}`}
            >
              {conflictCount > 0 ? `${conflictCount} Conflicts` : "All Clean"}
            </span>
          </div>
          <p className="mt-3 text-xs font-medium text-slate-500 dark:text-slate-400">
            {conflictCount > 0
              ? "Resolve conflicts now →"
              : "Discover new dotfiles →"}
          </p>
        </div>
      </div>

      {/* Sync Log & Error Panel */}
      {(() => {
        const syncRes = lastSyncResult || status.last_sync;
        if (!syncRes) return null;

        const hasErrors = syncRes.errors && syncRes.errors.length > 0;

        return (
          <div
            className={`rounded-xl border p-5 shadow-sm transition-all ${
              hasErrors
                ? "bg-rose-50/50 dark:bg-rose-950/20 border-rose-200 dark:border-rose-900/60"
                : "bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800"
            }`}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                {hasErrors ? (
                  <AlertCircle className="w-5 h-5 text-rose-600 dark:text-rose-400" />
                ) : (
                  <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                )}
                <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                  {hasErrors
                    ? `Sync Issues Encountered (${syncRes.errors.length} error${syncRes.errors.length > 1 ? "s" : ""})`
                    : "Last Sync Summary"}
                </h3>
              </div>
              <span className="text-xs text-slate-500 font-mono">
                {new Date(syncRes.timestamp).toLocaleTimeString()}
              </span>
            </div>

            <p className="text-xs text-slate-600 dark:text-slate-400 mb-3">
              {syncRes.message}
            </p>

            {/* Error list */}
            {hasErrors && (
              <div className="space-y-2 mt-2">
                <span className="text-xs font-semibold text-rose-700 dark:text-rose-300 block">
                  Error Details:
                </span>
                <div className="bg-rose-100/60 dark:bg-slate-950 rounded-lg p-3 border border-rose-200 dark:border-rose-900/50 font-mono text-xs space-y-1.5 text-rose-800 dark:text-rose-300 overflow-x-auto">
                  {syncRes.errors.map((errStr, idx) => (
                    <div key={idx} className="flex items-start gap-2">
                      <span className="text-rose-500 shrink-0 font-bold">
                        •
                      </span>
                      <span className="break-all">{errStr}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Successfully applied files */}
            {syncRes.applied_files && syncRes.applied_files.length > 0 && (
              <div className="mt-3 text-xs text-slate-500 dark:text-slate-400">
                <span className="font-semibold text-slate-700 dark:text-slate-300">
                  Applied Dotfiles ({syncRes.applied_files.length}):
                </span>{" "}
                {syncRes.applied_files.map((f) => `~/${f}`).join(", ")}
              </div>
            )}
          </div>
        );
      })()}

      {/* Details & Architecture Banner */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 shadow-sm">
        <h3 className="text-base font-semibold text-slate-900 dark:text-white mb-4">
          Environment & Storage Architecture
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
          <div className="p-4 rounded-lg bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400 block mb-1">
              Local Git Storage Directory
            </span>
            <code className="text-xs font-mono text-indigo-600 dark:text-indigo-400 break-all">
              {config.storage_dir}
            </code>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-2">
              Dotfiles are safely centralized here and symlinked to{" "}
              <code className="font-mono text-xs">$HOME</code>.
            </p>
          </div>

          <div className="p-4 rounded-lg bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400 block mb-1">
              Automated Safety Backups
            </span>
            <code className="text-xs font-mono text-indigo-600 dark:text-indigo-400 break-all">
              {config.backup_dir}
            </code>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-2">
              Every file modified or linked is timestamp-snapshotted for instant
              recovery.
            </p>
          </div>
        </div>

        {/* Remote setup prompt if empty */}
        {!isGitConfigured && (
          <div className="mt-4 p-4 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/50 flex items-start justify-between">
            <div>
              <h4 className="text-sm font-semibold text-amber-800 dark:text-amber-300">
                Git Remote Not Configured
              </h4>
              <p className="text-xs text-amber-700 dark:text-amber-400 mt-0.5">
                Connect your GitHub or GitLab repository to synchronize dotfiles
                across multiple computers.
              </p>
            </div>
            <button
              onClick={() => onSwitchTab("settings")}
              className="text-xs px-3 py-1.5 rounded-lg font-medium bg-amber-600 hover:bg-amber-500 text-white shrink-0 ml-4 transition-colors"
            >
              Configure Git →
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
