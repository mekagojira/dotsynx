export type SyncMode = "manual" | "auto" | "boot";
export type ConflictStrategy = "interactive" | "ours" | "theirs";

export interface TrackedItem {
  path: string;
  repo_path?: string;
  is_dir: boolean;
  enabled: boolean;
  description?: string;
}

export interface Config {
  repo_url: string;
  branch: string;
  storage_dir: string;
  backup_dir: string;
  sync_mode: SyncMode;
  interval: string;
  conflict_strategy: ConflictStrategy;
  theme: string;
  tracked: TrackedItem[];
  web_port: number;
}

export interface GitStatus {
  Initialized: boolean;
  Branch: string;
  RemoteURL: string;
  Ahead: number;
  Behind: number;
  Clean: boolean;
  HasConflict: boolean;
  Modified: string[];
  Untracked: string[];
}

export interface ServiceStatus {
  installed: boolean;
  running: boolean;
  pid?: number;
  platform: string;
  service_file: string;
  message: string;
}

export interface DaemonStatus {
  running: boolean;
  pid: number;
}

export interface ConflictItem {
  path: string;
  diff: string;
  repo_path: string;
  home_path: string;
  can_auto_fix: boolean;
}

export interface SystemStatus {
  git: GitStatus;
  daemon: DaemonStatus;
  service?: ServiceStatus;
  conflicts: ConflictItem[];
  config: Config;
  last_sync?: SyncResult;
  sync_logs?: SyncResult[];
}

export type FileLinkStatus =
  "linked" | "broken" | "unlinked" | "missing" | "untracked";

export interface ItemStatus {
  item: TrackedItem;
  home_abs_path: string;
  repo_abs_path: string;
  link_status: FileLinkStatus;
  home_exists: boolean;
  repo_exists: boolean;
  has_diff: boolean;
  error_message?: string;
}

export interface Suggestion {
  path: string;
  abs_path: string;
  is_dir: boolean;
  category: string;
  description: string;
  exists: boolean;
  is_sensitive: boolean;
  size_human?: string;
}

export interface BrowseEntry {
  name: string;
  rel_home: string;
  abs_path: string;
  is_dir: boolean;
  is_sensitive: boolean;
  is_tracked: boolean;
  size: number;
}

export interface BrowseResponse {
  current_path: string;
  parent_path: string;
  home_path: string;
  entries: BrowseEntry[];
}

export interface BackupEntry {
  id: string;
  timestamp: string;
  path: string;
  file_count: number;
}

export interface LogEntry {
  timestamp: string;
  level: "INFO" | "WARN" | "ERROR" | "SYNC";
  message: string;
  raw: string;
}

export interface SyncResult {
  success: boolean;
  pulled: boolean;
  pushed: boolean;
  local_changes: number;
  applied_files: string[];
  errors: string[];
  conflicts: ConflictItem[];
  timestamp: string;
  message: string;
}
