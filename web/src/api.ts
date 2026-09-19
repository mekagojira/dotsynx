import {
  SystemStatus,
  Config,
  ItemStatus,
  Suggestion,
  BrowseResponse,
  BackupEntry,
  SyncResult,
  ConflictItem,
  LogEntry,
} from "./types";

const API_BASE = "/api";

export async function fetchStatus(): Promise<SystemStatus> {
  const res = await fetch(`${API_BASE}/status`);
  if (!res.ok) throw new Error(`Status failed: ${res.statusText}`);
  return res.json();
}

export async function triggerSync(): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/sync`, { method: "POST" });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || `Sync failed: ${res.statusText}`);
  }
  return res.json();
}

export async function resetHardToRemote(): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/reset-remote`, { method: "POST" });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || `Reset failed: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchConfig(): Promise<Config> {
  const res = await fetch(`${API_BASE}/config`);
  if (!res.ok) throw new Error("Failed to get config");
  return res.json();
}

export async function updateConfig(
  cfg: Partial<Config> & { reset_to_remote?: boolean },
): Promise<Config> {
  const res = await fetch(`${API_BASE}/config`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cfg),
  });
  if (!res.ok) throw new Error("Failed to update config");
  return res.json();
}

export async function fetchFiles(): Promise<ItemStatus[]> {
  const res = await fetch(`${API_BASE}/files`);
  if (!res.ok) throw new Error("Failed to fetch files");
  return res.json();
}

export async function trackItem(
  path: string,
  isDir: boolean,
  description = "",
): Promise<void> {
  const res = await fetch(`${API_BASE}/files/track`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, is_dir: isDir, description }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || "Failed to track item");
  }
}

export async function untrackItem(
  path: string,
  keepFile = true,
): Promise<void> {
  const res = await fetch(`${API_BASE}/files/untrack`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, keep_file: keepFile }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || "Failed to untrack item");
  }
}

export async function toggleItem(path: string): Promise<void> {
  const res = await fetch(`${API_BASE}/files/toggle`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  if (!res.ok) throw new Error("Failed to toggle item");
}

export async function fetchSuggestions(): Promise<Suggestion[]> {
  const res = await fetch(`${API_BASE}/suggestions`);
  if (!res.ok) throw new Error("Failed to fetch suggestions");
  return res.json();
}

export async function browsePath(path = ""): Promise<BrowseResponse> {
  const url = path
    ? `${API_BASE}/browse?path=${encodeURIComponent(path)}`
    : `${API_BASE}/browse`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("Failed to browse path");
  return res.json();
}

export async function fetchConflicts(): Promise<ConflictItem[]> {
  const res = await fetch(`${API_BASE}/conflicts`);
  if (!res.ok) throw new Error("Failed to fetch conflicts");
  return res.json();
}

export async function resolveConflict(
  path: string,
  strategy: "ours" | "theirs",
): Promise<void> {
  const res = await fetch(`${API_BASE}/conflicts/resolve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, strategy }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || "Failed to resolve conflict");
  }
}

export async function fetchBackups(): Promise<BackupEntry[]> {
  const res = await fetch(`${API_BASE}/backups`);
  if (!res.ok) throw new Error("Failed to fetch backups");
  return res.json();
}

export async function fetchLogs(lines = 200, since = ""): Promise<LogEntry[]> {
  const url = since
    ? `${API_BASE}/logs?lines=${lines}&since=${encodeURIComponent(since)}`
    : `${API_BASE}/logs?lines=${lines}`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("Failed to fetch logs");
  return res.json();
}

export async function startDaemon(): Promise<void> {
  const res = await fetch(`${API_BASE}/daemon/start`, { method: "POST" });
  if (!res.ok) throw new Error("Failed to start daemon");
}

export async function stopDaemon(): Promise<void> {
  const res = await fetch(`${API_BASE}/daemon/stop`, { method: "POST" });
  if (!res.ok) throw new Error("Failed to stop daemon");
}
