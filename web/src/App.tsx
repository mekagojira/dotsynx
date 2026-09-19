import React, { useState, useEffect } from "react";
import {
  LayoutDashboard,
  Files,
  Sparkles,
  FolderSearch,
  AlertTriangle,
  Archive,
  ScrollText,
  Settings,
} from "lucide-react";
import { Header } from "./components/Header";
import { DashboardTab } from "./components/DashboardTab";
import { TrackedFilesTab } from "./components/TrackedFilesTab";
import { SuggestionsTab } from "./components/SuggestionsTab";
import { BrowserTab } from "./components/BrowserTab";
import { ConflictsTab } from "./components/ConflictsTab";
import { BackupsTab } from "./components/BackupsTab";
import { LogsTab } from "./components/LogsTab";
import { SettingsTab } from "./components/SettingsTab";
import {
  SystemStatus,
  ItemStatus,
  Suggestion,
  SyncMode,
  Config,
  SyncResult,
} from "./types";
import {
  fetchStatus,
  triggerSync,
  fetchFiles,
  trackItem,
  untrackItem,
  toggleItem,
  fetchSuggestions,
  resolveConflict,
  updateConfig,
  resetHardToRemote,
  startDaemon,
  stopDaemon,
} from "./api";

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState("dashboard");
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [files, setFiles] = useState<ItemStatus[]>([]);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [lastSyncResult, setLastSyncResult] = useState<SyncResult | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [bannerMsg, setBannerMsg] = useState<string>("");
  const [isError, setIsError] = useState(false);

  const loadData = async () => {
    try {
      const [st, fl, sg] = await Promise.all([
        fetchStatus(),
        fetchFiles(),
        fetchSuggestions(),
      ]);
      setStatus(st);
      setFiles(Array.isArray(fl) ? fl : []);
      setSuggestions(Array.isArray(sg) ? sg : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 10000); // 10s poll
    return () => clearInterval(interval);
  }, []);

  const showNotification = (msg: string, err = false) => {
    setBannerMsg(msg);
    setIsError(err);
    setTimeout(() => {
      setBannerMsg("");
      setIsError(false);
    }, 6000);
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const res = await triggerSync();
      setLastSyncResult(res);
      showNotification(res.message, !res.success);
      if (!res.success) {
        setActiveTab("dashboard");
      }
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Sync failed", true);
    } finally {
      setSyncing(false);
    }
  };

  const handleTrack = async (path: string, isDir: boolean, desc: string) => {
    try {
      await trackItem(path, isDir, desc);
      showNotification(`Added ~/${path} to tracking`);
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Failed to track item", true);
    }
  };

  const handleUntrack = async (path: string, keepFile: boolean) => {
    try {
      await untrackItem(path, keepFile);
      showNotification(`Untracked ~/${path}`);
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Failed to untrack", true);
    }
  };

  const handleToggle = async (path: string) => {
    try {
      await toggleItem(path);
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Failed to toggle item", true);
    }
  };

  const handleResolve = async (path: string, strategy: "ours" | "theirs") => {
    try {
      await resolveConflict(path, strategy);
      showNotification(
        `Resolved conflict in ${path} using ${strategy} version`,
      );
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Failed to resolve conflict", true);
    }
  };

  const handleSaveConfig = async (
    updated: Partial<Config> & { reset_to_remote?: boolean },
  ) => {
    try {
      await updateConfig(updated);
      showNotification(
        updated.reset_to_remote
          ? "Settings saved & local storage reset to remote!"
          : "Settings saved successfully",
      );
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Failed to save settings", true);
    }
  };

  const handleResetToRemote = async () => {
    setSyncing(true);
    try {
      const res = await resetHardToRemote();
      setLastSyncResult(res);
      showNotification(res.message, !res.success);
      await loadData();
    } catch (err: any) {
      showNotification(err.message || "Failed to reset to remote", true);
    } finally {
      setSyncing(false);
    }
  };

  const trackedPathsSet = new Set(
    (status?.config?.tracked || []).map((t) => t.path),
  );
  const conflictCount = status?.conflicts?.length || 0;

  const tabs = [
    { id: "dashboard", label: "Dashboard", icon: LayoutDashboard },
    {
      id: "files",
      label: "Tracked Files",
      icon: Files,
      badge: (files || []).length,
    },
    {
      id: "suggestions",
      label: "Suggestions",
      icon: Sparkles,
      badge: (suggestions || []).length,
    },
    { id: "browser", label: "Browse $HOME", icon: FolderSearch },
    {
      id: "conflicts",
      label: "Conflicts",
      icon: AlertTriangle,
      badge: conflictCount,
      badgeDanger: conflictCount > 0,
    },
    { id: "logs", label: "Logs", icon: ScrollText },
    { id: "backups", label: "Backups", icon: Archive },
    { id: "settings", label: "Settings", icon: Settings },
  ];

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 flex flex-col font-sans transition-colors duration-200">
      <Header
        status={status}
        syncing={syncing}
        onSync={handleSync}
        lastMessage={bannerMsg}
        isError={isError}
      />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 w-full flex-1">
        {/* Navigation Tabs */}
        <div className="flex items-center gap-1.5 border-b border-slate-200 dark:border-slate-800 mb-6 overflow-x-auto pb-2">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all shrink-0 ${
                  isActive
                    ? "bg-indigo-600 text-white shadow-sm shadow-indigo-600/20"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-900"
                }`}
              >
                <Icon className="w-4 h-4" />
                <span>{tab.label}</span>
                {typeof tab.badge === "number" && tab.badge > 0 && (
                  <span
                    className={`text-[10px] px-1.5 py-0.2 rounded-full font-bold ${
                      tab.badgeDanger
                        ? "bg-rose-500 text-white"
                        : isActive
                          ? "bg-indigo-700 text-white"
                          : "bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300"
                    }`}
                  >
                    {tab.badge}
                  </span>
                )}
              </button>
            );
          })}
        </div>

        {/* Tab views */}
        <main>
          {activeTab === "dashboard" && (
            <DashboardTab
              status={status}
              lastSyncResult={lastSyncResult}
              onUpdateMode={(mode: SyncMode) =>
                handleSaveConfig({ sync_mode: mode })
              }
              onSwitchTab={setActiveTab}
            />
          )}

          {activeTab === "files" && (
            <TrackedFilesTab
              files={files}
              onToggle={handleToggle}
              onUntrack={handleUntrack}
              onAdd={handleTrack}
              onRefresh={loadData}
            />
          )}

          {activeTab === "suggestions" && (
            <SuggestionsTab
              suggestions={suggestions}
              trackedPaths={trackedPathsSet}
              onTrack={handleTrack}
            />
          )}

          {activeTab === "browser" && (
            <BrowserTab onTrack={handleTrack} onUntrack={handleUntrack} />
          )}

          {activeTab === "conflicts" && (
            <ConflictsTab
              conflicts={status?.conflicts || []}
              onResolve={handleResolve}
            />
          )}

          {activeTab === "logs" && <LogsTab />}

          {activeTab === "backups" && <BackupsTab />}

          {activeTab === "settings" && status && (
            <SettingsTab
              config={status.config}
              onSave={handleSaveConfig}
              onResetToRemote={handleResetToRemote}
              onStartDaemon={async () => {
                await startDaemon();
                loadData();
              }}
              onStopDaemon={async () => {
                await stopDaemon();
                loadData();
              }}
              daemonRunning={status.daemon.running}
            />
          )}
        </main>
      </div>
    </div>
  );
};
