import type { ColumnConfig } from '../types/app';

export const ALL_COLUMNS: Record<string, ColumnConfig> = {
  MSISDN: { name: 'MSISDN', field: 'MSISDN', sortable: true, sortKey: 'MSISDN', default: true },
  CLI: { name: 'CLI', field: 'CLI', sortable: true, sortKey: 'CLI', default: true },
  SIM_STATUS_CHANGE: { name: 'Status', field: 'SIM_STATUS_CHANGE', sortable: true, sortKey: 'SIM_STATUS_CHANGE', default: true },
  RATE_PLAN: { name: 'Rate Plan', field: 'RATE_PLAN_FULL_NAME', sortable: true, sortKey: 'RATE_PLAN_CHANGE', default: true },
  CUSTOMER_LABEL_1: { name: 'Label 1', field: 'CUSTOMER_LABEL_1', sortable: true, sortKey: 'CUSTOMER_LABEL_1', default: true },
  CUSTOMER_LABEL_2: { name: 'Label 2', field: 'CUSTOMER_LABEL_2', sortable: true, sortKey: 'CUSTOMER_LABEL_2', default: false },
  CUSTOMER_LABEL_3: { name: 'Label 3', field: 'CUSTOMER_LABEL_3', sortable: true, sortKey: 'CUSTOMER_LABEL_3', default: false },
  SIM_SWAP: { name: 'ICCID', field: 'SIM_SWAP', sortable: false, default: false },
  IMSI: { name: 'IMSI', field: 'IMSI', sortable: false, default: false },
  IMEI: { name: 'IMEI', field: 'IMEI', sortable: false, default: false },
  APN_NAME: { name: 'APN', field: 'APN_NAME', sortable: true, sortKey: 'APN_NAME', default: false },
  IP1: { name: 'IP Address', field: 'IP1', sortable: false, default: false },
  MONTHLY_USAGE_MB: { name: 'Usage (MB)', field: 'MONTHLY_USAGE_MB', sortable: true, sortKey: 'MONTHLY_USAGE_MB', default: true },
  ALLOCATED_MB: { name: 'Allocated (MB)', field: 'ALLOCATED_MB', sortable: true, sortKey: 'ALLOCATED_MB', default: false },
  LAST_SESSION_TIME: { name: 'Last Session', field: 'LAST_SESSION_TIME', sortable: true, sortKey: 'LAST_SESSION_TIME', default: false },
  IN_SESSION: { name: 'In Session', field: 'IN_SESSION', sortable: false, default: false }
};

export const STORAGE_KEYS = {
  columns: 'eyeson_visible_columns',
  columnOrder: 'eyeson_column_order',
  session: 'eyeson_session',
  theme: 'theme'
} as const;

export const SESSION_DURATION_MS = 24 * 60 * 60 * 1000;
