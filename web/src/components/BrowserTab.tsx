import React, { useState, useEffect } from 'react';
import { Folder, FileText, ArrowLeft, Plus, Check, ShieldAlert, HardDrive, RefreshCw } from 'lucide-react';
import { BrowseResponse, BrowseEntry } from '../types';
import { browsePath } from '../api';

interface BrowserTabProps {
  onTrack: (path: string, isDir: boolean, desc: string) => void;
  onUntrack: (path: string, keepFile: boolean) => void;
}

export const BrowserTab: React.FC<BrowserTabProps> = ({ onTrack, onUntrack }) => {
  const [data, setData] = useState<BrowseResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [currentPath, setCurrentPath] = useState('');
  const [showHidden, setShowHidden] = useState(true);

  const loadDirectory = async (path: string) => {
    setLoading(true);
    try {
      const res = await browsePath(path);
      setData(res);
      setCurrentPath(res.current_path);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDirectory('');
  }, []);

  if (!data && loading) {
    return (
      <div className="py-12 text-center text-slate-400">
        Loading directory browser...
      </div>
    );
  }

  const entries = (data?.entries || []).filter(e => {
    if (!showHidden && e.name.startsWith('.')) return false;
    return true;
  });

  return (
    <div className="space-y-4">
      {/* Navigation bar */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <button
            onClick={() => loadDirectory(data?.parent_path || '')}
            disabled={!data || data.current_path === data.home_path}
            className="p-2 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed text-slate-600 dark:text-slate-300"
            title="Go up"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>

          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 min-w-0 text-sm font-mono truncate">
            <HardDrive className="w-4 h-4 text-indigo-500 shrink-0" />
            <span className="truncate text-slate-700 dark:text-slate-300">{currentPath}</span>
          </div>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 cursor-pointer">
            <input
              type="checkbox"
              checked={showHidden}
              onChange={(e) => setShowHidden(e.target.checked)}
              className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <span>Show hidden dotfiles (.)</span>
          </label>

          <button
            onClick={() => loadDirectory(currentPath)}
            className="p-2 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 text-slate-500"
            title="Reload"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Directory entries table */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-sm">
        <div className="divide-y divide-slate-100 dark:divide-slate-800">
          {entries.length === 0 ? (
            <div className="p-8 text-center text-slate-400">
              Folder is empty or files are hidden.
            </div>
          ) : (
            entries.map((entry) => (
              <div
                key={entry.name}
                className="p-3.5 px-4 flex items-center justify-between hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
              >
                <div
                  className="flex items-center gap-3 min-w-0 cursor-pointer flex-1"
                  onClick={() => {
                    if (entry.is_dir) {
                      loadDirectory(entry.abs_path);
                    }
                  }}
                >
                  <div className="p-1.5 rounded bg-slate-100 dark:bg-slate-800 shrink-0">
                    {entry.is_dir ? (
                      <Folder className="w-4 h-4 text-sky-500" />
                    ) : (
                      <FileText className="w-4 h-4 text-indigo-500" />
                    )}
                  </div>
                  <span className="font-mono text-sm text-slate-800 dark:text-slate-200 truncate hover:text-indigo-600 dark:hover:text-indigo-400">
                    {entry.name}
                  </span>

                  {entry.is_sensitive && (
                    <span className="flex items-center gap-1 text-[11px] text-rose-600 dark:text-rose-400 bg-rose-50 dark:bg-rose-950/50 px-2 py-0.5 rounded border border-rose-200 dark:border-rose-900">
                      <ShieldAlert className="w-3 h-3 shrink-0" />
                      <span>Private / Sensitive</span>
                    </span>
                  )}
                </div>

                <div className="flex items-center gap-3 shrink-0 ml-3">
                  {entry.is_tracked ? (
                    <button
                      onClick={() => onUntrack(entry.rel_home, true)}
                      className="flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800 hover:bg-rose-50 hover:text-rose-700 hover:border-rose-300 dark:hover:bg-rose-950 dark:hover:text-rose-300 transition-colors"
                      title="Click to untrack"
                    >
                      <Check className="w-3 h-3" />
                      <span>Tracked</span>
                    </button>
                  ) : (
                    <button
                      onClick={() => onTrack(entry.rel_home, entry.is_dir, '')}
                      className="flex items-center gap-1 px-2.5 py-1 rounded text-xs font-medium bg-slate-100 hover:bg-indigo-50 hover:text-indigo-700 dark:bg-slate-800 dark:hover:bg-indigo-950/60 dark:hover:text-indigo-300 text-slate-600 dark:text-slate-300 transition-colors"
                    >
                      <Plus className="w-3 h-3" />
                      <span>Track {entry.is_dir ? 'Dir' : 'File'}</span>
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
