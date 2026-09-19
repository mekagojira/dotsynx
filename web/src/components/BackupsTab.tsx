import React, { useEffect, useState } from 'react';
import { Archive, Clock, Files, RefreshCw, Folder } from 'lucide-react';
import { BackupEntry } from '../types';
import { fetchBackups } from '../api';

export const BackupsTab: React.FC = () => {
  const [backups, setBackups] = useState<BackupEntry[]>([]);
  const [loading, setLoading] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const list = await fetchBackups();
      setBackups(list);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between bg-white dark:bg-slate-900 p-4 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <div>
          <h3 className="text-base font-bold text-slate-900 dark:text-white">
            Safety Snapshots & Backups
          </h3>
          <p className="text-xs text-slate-500">
            dotsynx preserves untouched copies of your dotfiles before every symlink, replace, or conflict resolution.
          </p>
        </div>

        <button
          onClick={load}
          className="p-2 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 text-slate-500"
          title="Refresh backups"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-sm">
        {backups.length === 0 ? (
          <div className="p-12 text-center text-slate-400">
            No backups recorded yet. Snapshots are created automatically before file overwrites.
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-800">
            {backups.map((b) => (
              <div
                key={b.id}
                className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-slate-50/50 dark:hover:bg-slate-800/30"
              >
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 shrink-0">
                    <Archive className="w-5 h-5 text-indigo-500" />
                  </div>
                  <div>
                    <span className="font-mono text-sm font-semibold text-slate-900 dark:text-white">
                      Snapshot {b.id}
                    </span>
                    <p className="text-xs font-mono text-slate-400 dark:text-slate-500 mt-0.5">
                      {b.path}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-4 text-xs text-slate-500 self-end sm:self-center">
                  <span className="flex items-center gap-1">
                    <Files className="w-3.5 h-3.5 text-slate-400" /> {b.file_count} files
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5 text-slate-400" />
                    {new Date(b.timestamp).toLocaleString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
