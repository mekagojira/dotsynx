import React, { useState } from 'react';
import { Save, GitBranch, Cpu, Clock, Terminal, AlertCircle } from 'lucide-react';
import { Config, SyncMode, ConflictStrategy } from '../types';

interface SettingsTabProps {
  config: Config;
  onSave: (updated: Partial<Config>) => void;
  onStartDaemon: () => void;
  onStopDaemon: () => void;
  daemonRunning: boolean;
}

export const SettingsTab: React.FC<SettingsTabProps> = ({
  config,
  onSave,
  onStartDaemon,
  onStopDaemon,
  daemonRunning,
}) => {
  const [repoUrl, setRepoUrl] = useState(config.repo_url || '');
  const [branch, setBranch] = useState(config.branch || 'main');
  const [syncMode, setSyncMode] = useState<SyncMode>(config.sync_mode || 'manual');
  const [interval, setInterval] = useState(config.interval || '15m');
  const [strategy, setStrategy] = useState<ConflictStrategy>(config.conflict_strategy || 'interactive');
  const [saved, setSaved] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      repo_url: repoUrl.trim(),
      branch: branch.trim(),
      sync_mode: syncMode,
      interval: interval.trim(),
      conflict_strategy: strategy,
    });
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  return (
    <div className="max-w-3xl space-y-6">
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Git settings */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 shadow-sm space-y-4">
          <div className="flex items-center gap-2 font-bold text-slate-900 dark:text-white">
            <GitBranch className="w-5 h-5 text-indigo-500" />
            <h3 className="text-base">Git Remote Repository</h3>
          </div>
          <p className="text-xs text-slate-500">
            Provide your remote repository URL (SSH or HTTPS). Works with public and private repositories using your system's SSH keys or Git credentials.
          </p>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Remote URL
            </label>
            <input
              type="text"
              placeholder="git@github.com:username/dotfiles.git or https://..."
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              className="w-full px-3.5 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Git Branch
            </label>
            <input
              type="text"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              className="w-full sm:w-48 px-3.5 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
            />
          </div>
        </div>

        {/* Sync Mode & Schedule */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 shadow-sm space-y-4">
          <div className="flex items-center gap-2 font-bold text-slate-900 dark:text-white">
            <Clock className="w-5 h-5 text-indigo-500" />
            <h3 className="text-base">Sync Mode & Triggers</h3>
          </div>
          <p className="text-xs text-slate-500">
            Choose when and how dotsynx synchronizes your dotfiles across machines.
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {[
              { id: 'manual', title: 'Manual', desc: 'Sync only on demand via CLI, TUI, or Web UI' },
              { id: 'auto', title: 'Automatic', desc: 'Sync automatically on a background interval or cron' },
              { id: 'boot', title: 'At Boot / Login', desc: 'Sync once whenever your system boots or you log in' },
            ].map((m) => (
              <div
                key={m.id}
                onClick={() => setSyncMode(m.id as SyncMode)}
                className={`p-4 rounded-xl border cursor-pointer transition-all ${
                  syncMode === m.id
                    ? 'border-indigo-600 bg-indigo-50/50 dark:bg-indigo-950/30'
                    : 'border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700'
                }`}
              >
                <div className="font-semibold text-sm text-slate-900 dark:text-white">
                  {m.title}
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  {m.desc}
                </div>
              </div>
            ))}
          </div>

          {syncMode === 'auto' && (
            <div className="pt-2">
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Interval Duration or Cron Syntax
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="15m, 30m, 1h, or 0 * * * *"
                  value={interval}
                  onChange={(e) => setInterval(e.target.value)}
                  className="w-64 px-3.5 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
                />
                <span className="text-xs text-slate-400">
                  e.g. "15m", "1h", or standard cron
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Conflict Resolution Strategy */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 shadow-sm space-y-4">
          <h3 className="text-base font-bold text-slate-900 dark:text-white">
            Default Conflict Resolution
          </h3>
          <p className="text-xs text-slate-500">
            How dotsynx resolves unexpected divergences during automated syncs.
          </p>

          <div className="flex flex-col sm:flex-row gap-3">
            {[
              { id: 'interactive', label: 'Interactive (Ask User)', desc: 'Show visual diff and pause until resolved' },
              { id: 'ours', label: 'Ours (Local Wins)', desc: 'Keep local machine modifications and overwrite remote' },
              { id: 'theirs', label: 'Theirs (Remote Wins)', desc: 'Pull remote changes and backup local version' },
            ].map((s) => (
              <label
                key={s.id}
                className={`flex-1 p-3.5 rounded-xl border cursor-pointer ${
                  strategy === s.id
                    ? 'border-indigo-600 bg-indigo-50/50 dark:bg-indigo-950/30'
                    : 'border-slate-200 dark:border-slate-800'
                }`}
              >
                <div className="flex items-center gap-2">
                  <input
                    type="radio"
                    name="strategy"
                    checked={strategy === s.id}
                    onChange={() => setStrategy(s.id as ConflictStrategy)}
                    className="text-indigo-600"
                  />
                  <span className="text-sm font-semibold text-slate-900 dark:text-white">{s.label}</span>
                </div>
                <p className="text-xs text-slate-500 mt-1 pl-5">{s.desc}</p>
              </label>
            ))}
          </div>
        </div>

        {/* Save button */}
        <div className="flex items-center justify-between">
          <div>
            {saved && (
              <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400">
                ✓ Configuration saved successfully!
              </span>
            )}
          </div>
          <button
            type="submit"
            className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-sm shadow-indigo-600/20 transition-colors"
          >
            <Save className="w-4 h-4" />
            <span>Save Configuration</span>
          </button>
        </div>
      </form>
    </div>
  );
};
