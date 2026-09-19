import React, { useState } from "react";
import {
  FileText,
  Folder,
  Check,
  X,
  Trash2,
  Plus,
  AlertCircle,
  RefreshCw,
  ExternalLink,
} from "lucide-react";
import { ItemStatus } from "../types";

interface TrackedFilesTabProps {
  files: ItemStatus[];
  onToggle: (path: string) => void;
  onUntrack: (path: string, keepFile: boolean) => void;
  onAdd: (path: string, isDir: boolean, desc: string) => void;
  onRefresh: () => void;
}

export const TrackedFilesTab: React.FC<TrackedFilesTabProps> = ({
  files = [],
  onToggle,
  onUntrack,
  onAdd,
  onRefresh,
}) => {
  const [showAddModal, setShowAddModal] = useState(false);
  const [newPath, setNewPath] = useState("");
  const [newIsDir, setNewIsDir] = useState(false);
  const [newDesc, setNewDesc] = useState("");
  const [filter, setFilter] = useState("");

  const safeFiles = files || [];
  const filteredFiles = safeFiles.filter(
    (f) =>
      f?.item?.path?.toLowerCase().includes(filter.toLowerCase()) ||
      (f?.item?.description &&
        f.item.description.toLowerCase().includes(filter.toLowerCase())),
  );

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPath.trim()) return;
    onAdd(newPath.trim(), newIsDir, newDesc.trim());
    setNewPath("");
    setNewDesc("");
    setShowAddModal(false);
  };

  const getStatusBadge = (status: ItemStatus) => {
    if (!status.item.enabled) {
      return (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
          Disabled
        </span>
      );
    }
    switch (status.link_status) {
      case "linked":
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800">
            <Check className="w-3 h-3" /> Linked
          </span>
        );
      case "broken":
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-rose-50 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300 border border-rose-200 dark:border-rose-800">
            <AlertCircle className="w-3 h-3" /> Broken Link
          </span>
        );
      case "unlinked":
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-amber-50 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300 border border-amber-200 dark:border-amber-800">
            Unlinked Regular File
          </span>
        );
      case "missing":
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-purple-50 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300 border border-purple-200 dark:border-purple-800">
            Missing in $HOME
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
            {status.link_status}
          </span>
        );
    }
  };

  return (
    <div className="space-y-4">
      {/* Controls row */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-white dark:bg-slate-900 p-4 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <input
          type="text"
          placeholder="Filter tracked dotfiles..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="px-3.5 py-1.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 w-full sm:w-72"
        />

        <div className="flex items-center gap-2">
          <button
            onClick={onRefresh}
            title="Refresh status"
            className="p-2 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          <button
            onClick={() => setShowAddModal(true)}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-sm transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>Add Dotfile / Folder</span>
          </button>
        </div>
      </div>

      {/* Files list */}
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-sm">
        {filteredFiles.length === 0 ? (
          <div className="p-12 text-center text-slate-400 dark:text-slate-500">
            {files.length === 0 ? (
              <div>
                <p className="font-medium text-slate-700 dark:text-slate-300">
                  No dotfiles tracked yet.
                </p>
                <p className="text-xs text-slate-500 mt-1">
                  Add files manually above or check the "Suggestions" tab for
                  detected tools.
                </p>
              </div>
            ) : (
              <p>No dotfiles match "{filter}"</p>
            )}
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-800">
            {filteredFiles.map((file) => (
              <div
                key={file.item.path}
                className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-slate-50/70 dark:hover:bg-slate-800/40 transition-colors"
              >
                <div className="flex items-start gap-3 min-w-0">
                  <div className="p-2 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 shrink-0">
                    {file.item.is_dir ? (
                      <Folder className="w-5 h-5 text-sky-500" />
                    ) : (
                      <FileText className="w-5 h-5 text-indigo-500" />
                    )}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-mono text-sm font-medium text-slate-900 dark:text-white truncate">
                        ~/{file.item.path}
                      </span>
                      {getStatusBadge(file)}
                    </div>
                    {file.item.description && (
                      <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                        {file.item.description}
                      </p>
                    )}
                    <p className="text-xs font-mono text-slate-400 dark:text-slate-500 mt-0.5 truncate">
                      Repo: {file.repo_abs_path}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3 shrink-0 self-end sm:self-center">
                  {/* Enable/Disable Toggle */}
                  <label className="flex items-center gap-2 cursor-pointer text-xs text-slate-600 dark:text-slate-400">
                    <span>{file.item.enabled ? "Syncing" : "Paused"}</span>
                    <input
                      type="checkbox"
                      checked={file.item.enabled}
                      onChange={() => onToggle(file.item.path)}
                      className="sr-only peer"
                    />
                    <div className="w-9 h-5 bg-slate-200 peer-focus:outline-none rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600 relative"></div>
                  </label>

                  {/* Untrack Button */}
                  <button
                    onClick={() => {
                      if (
                        confirm(
                          `Untrack ~/.${file.item.path}? Your local file will be preserved.`,
                        )
                      ) {
                        onUntrack(file.item.path, true);
                      }
                    }}
                    title="Stop tracking this item"
                    className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Add Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 dark:border-slate-800">
            <h3 className="text-lg font-bold text-slate-900 dark:text-white mb-2">
              Track a Dotfile or Directory
            </h3>
            <p className="text-xs text-slate-500 mb-4">
              Enter the path relative to your home directory (e.g.,{" "}
              <code className="font-mono">.zshrc</code> or{" "}
              <code className="font-mono">.config/nvim</code>).
            </p>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Path in $HOME
                </label>
                <div className="flex items-center">
                  <span className="px-3 py-2 bg-slate-100 dark:bg-slate-800 text-slate-500 rounded-l-lg border border-r-0 border-slate-200 dark:border-slate-700 text-sm font-mono">
                    ~/
                  </span>
                  <input
                    type="text"
                    required
                    placeholder=".config/my-app"
                    value={newPath}
                    onChange={(e) => setNewPath(e.target.value)}
                    className="flex-1 px-3 py-2 rounded-r-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Description (optional)
                </label>
                <input
                  type="text"
                  placeholder="My favorite editor config"
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
                />
              </div>

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="isDirCheck"
                  checked={newIsDir}
                  onChange={(e) => setNewIsDir(e.target.checked)}
                  className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                />
                <label
                  htmlFor="isDirCheck"
                  className="text-sm text-slate-700 dark:text-slate-300"
                >
                  This is a directory / folder
                </label>
              </div>

              <div className="flex justify-end gap-3 pt-3">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors shadow-sm"
                >
                  Start Tracking
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
