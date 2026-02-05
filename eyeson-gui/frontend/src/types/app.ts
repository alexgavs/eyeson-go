export interface PendingJob {
  requestId: number;
  msisdns: string[];
  targetStatus: string;
  startTime: number;
  attempts: number;
}

export interface PendingStatus {
  msisdn: string;
  targetStatus: string;
  attempts: number;
  startTime: number;
}

export interface ColumnConfig {
  name: string;
  field: string;
  sortable: boolean;
  sortKey?: string;
  default: boolean;
}

export interface Toast {
  id: number;
  message: string;
  type: 'info' | 'success' | 'warning' | 'danger';
}

export interface SessionData {
  token: string;
  username: string;
  expiresAt: number;
}

export type NavPage = 'sims' | 'jobs' | 'stats' | 'admin' | 'profile' | 'queue';
