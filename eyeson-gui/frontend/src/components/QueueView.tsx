import React, { useEffect, useState, useMemo } from 'react';
import { ExecuteQueueTask, GetSyncQueue } from '../api';
import { formatDate } from '../utils/format';

// Парсинг payload и форматирование деталей задачи
function formatTaskDetails(task: any) {
  // Используем поля напрямую из task если они есть
  const msisdn = task.target_msisdn || task.target_cli || '';
  const cli = task.target_cli || '';
  const oldStatus = task.old_status || '';
  const newStatus = task.new_status || '';
  const type = task.type || '';

  // Также пробуем парсить payload
  let payload: any = {};
  try {
    if (task.payload) {
      payload = JSON.parse(task.payload);
    }
  } catch (e) {
    // ignore
  }

  // Приоритет: поля task, затем payload
  const displayMsisdn = msisdn || payload.msisdn || payload.msisdns?.[0] || cli || '';
  const displayOldStatus = oldStatus || payload.old_status || '—';
  const displayNewStatus = newStatus || payload.new_status || payload.status || '';
  const displayCli = cli || payload.cli || '';

  if (type === 'STATUS_CHANGE' || type === 'CHANGE_STATUS' || type === 'BULK_CHANGE') {
    return (
      <div className="d-flex flex-column gap-1">
        <div className="d-flex align-items-center gap-2">
          <span className="badge bg-secondary">📱</span>
          <span className="fw-medium">{displayCli || displayMsisdn}</span>
        </div>
        <div className="d-flex align-items-center gap-1">
          <span className={`badge ${getStatusBadgeClass(displayOldStatus)}`} style={{ minWidth: '70px' }}>
            {displayOldStatus}
          </span>
          <span className="text-muted">→</span>
          <span className={`badge ${getStatusBadgeClass(displayNewStatus)}`} style={{ minWidth: '70px' }}>
            {displayNewStatus}
          </span>
        </div>
      </div>
    );
  }

  if (type === 'LABEL_UPDATE' || type === 'UPDATE_SIM') {
    const field = task.label_field || payload.field || '';
    const value = task.label_value || payload.value || '';
    return (
      <div className="d-flex flex-column gap-1">
        <div className="d-flex align-items-center gap-2">
          <span className="badge bg-secondary">📱</span>
          <span className="fw-medium">{displayMsisdn}</span>
        </div>
        <div className="small">
          <span className="text-muted">{field}:</span>{' '}
          <span className="text-primary">{value}</span>
        </div>
      </div>
    );
  }

  // Fallback - показываем что есть
  if (displayMsisdn) {
    return (
      <div className="d-flex align-items-center gap-2">
        <span className="badge bg-secondary">📱</span>
        <span className="fw-medium">{displayMsisdn}</span>
      </div>
    );
  }

  // Если ничего не распарсили - показываем сокращённый payload
  const shortPayload = task.payload?.substring(0, 50) || '—';
  return <span className="text-muted small">{shortPayload}{task.payload?.length > 50 ? '...' : ''}</span>;
}

function getStatusBadgeClass(status: string): string {
  if (!status || status === '—') return 'bg-secondary';
  const s = status.toLowerCase();
  if (s.includes('activ')) return 'bg-success';
  if (s.includes('suspend')) return 'bg-warning text-dark';
  if (s.includes('terminat') || s.includes('deactiv')) return 'bg-danger';
  if (s.includes('test')) return 'bg-info text-dark';
  return 'bg-secondary';
}

type SortField = 'id' | 'type' | 'status' | 'attempts' | 'next_run_at' | 'created_at';
type SortDirection = 'asc' | 'desc';

export function QueueView() {
  const [tasks, setTasks] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [executing, setExecuting] = useState<number | null>(null);
  const [currentTime, setCurrentTime] = useState(Date.now());

  // Filters
  const [statusFilter, setStatusFilter] = useState<string>('ALL');
  const [typeFilter, setTypeFilter] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState<string>('');

  // Sorting
  const [sortField, setSortField] = useState<SortField>('id');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

  // Pagination
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const fetchQueue = async () => {
    if (tasks.length === 0) setLoading(true);
    try {
      const resp = await GetSyncQueue();
      setTasks(resp.data || []);
    } catch (e) {
      console.error(e);
    } finally {
      if (tasks.length === 0) setLoading(false);
    }
  };

  const executeTask = async (taskId: number) => {
    if (executing) return;

    setExecuting(taskId);
    try {
      await ExecuteQueueTask(taskId);
      await fetchQueue();
    } catch (e: any) {
      alert(`Error: ${e.message || 'Failed to execute task'}`);
    } finally {
      setExecuting(null);
    }
  };

  const getTimeUntil = (nextRunAt: string) => {
    const target = new Date(nextRunAt).getTime();
    const diff = target - currentTime;

    if (diff <= 0) return 'Now';

    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
  };

  // Get unique values for filters
  const uniqueStatuses = useMemo(() => {
    const statuses = new Set(tasks.map(t => t.status));
    return Array.from(statuses).sort();
  }, [tasks]);

  const uniqueTypes = useMemo(() => {
    const types = new Set(tasks.map(t => t.type));
    return Array.from(types).sort();
  }, [tasks]);

  // Filter, search, and sort tasks
  const filteredTasks = useMemo(() => {
    let result = [...tasks];

    // Status filter
    if (statusFilter !== 'ALL') {
      result = result.filter(t => t.status === statusFilter);
    }

    // Type filter
    if (typeFilter !== 'ALL') {
      result = result.filter(t => t.type === typeFilter);
    }

    // Search (by MSISDN, CLI, username, batch_id)
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(t => {
        const msisdn = (t.target_msisdn || '').toLowerCase();
        const cli = (t.target_cli || '').toLowerCase();
        const username = (t.username || '').toLowerCase();
        const batchId = (t.batch_id || '').toLowerCase();
        const payload = (t.payload || '').toLowerCase();
        return msisdn.includes(q) || cli.includes(q) || username.includes(q) || batchId.includes(q) || payload.includes(q);
      });
    }

    // Sort
    result.sort((a, b) => {
      let aVal = a[sortField];
      let bVal = b[sortField];

      // Handle dates
      if (sortField === 'next_run_at' || sortField === 'created_at') {
        aVal = aVal ? new Date(aVal).getTime() : 0;
        bVal = bVal ? new Date(bVal).getTime() : 0;
      }

      // Handle numbers
      if (sortField === 'id' || sortField === 'attempts') {
        aVal = Number(aVal) || 0;
        bVal = Number(bVal) || 0;
      }

      if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    return result;
  }, [tasks, statusFilter, typeFilter, searchQuery, sortField, sortDirection]);

  // Pagination
  const totalPages = Math.ceil(filteredTasks.length / pageSize);
  const paginatedTasks = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredTasks.slice(start, start + pageSize);
  }, [filteredTasks, page, pageSize]);

  // Reset page when filters change
  useEffect(() => {
    setPage(1);
  }, [statusFilter, typeFilter, searchQuery, pageSize]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(prev => prev === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return <span className="text-muted ms-1">↕</span>;
    return <span className="ms-1">{sortDirection === 'asc' ? '↑' : '↓'}</span>;
  };

  useEffect(() => {
    fetchQueue();
    const queueInterval = setInterval(fetchQueue, 2000);
    const timeInterval = setInterval(() => setCurrentTime(Date.now()), 1000);

    return () => {
      clearInterval(queueInterval);
      clearInterval(timeInterval);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="card shadow-sm">
      <div className="card-header bg-warning bg-opacity-10 d-flex justify-content-between align-items-center">
        <h5 className="mb-0 text-dark">⏳ Pending Confirmation Queue (Internal)</h5>
        <div className="d-flex gap-2 align-items-center">
          <span className="badge bg-secondary">{filteredTasks.length} / {tasks.length}</span>
          <button className="btn btn-sm btn-outline-secondary" onClick={fetchQueue}>
            Refresh
          </button>
        </div>
      </div>
      
      {/* Filters Bar */}
      <div className="card-body border-bottom py-2">
        <div className="row g-2 align-items-center">
          {/* Search */}
          <div className="col-md-4">
            <div className="input-group input-group-sm">
              <span className="input-group-text">🔍</span>
              <input
                type="text"
                className="form-control"
                placeholder="Search MSISDN, CLI, User..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              {searchQuery && (
                <button 
                  className="btn btn-outline-secondary" 
                  onClick={() => setSearchQuery('')}
                >
                  ✕
                </button>
              )}
            </div>
          </div>
          
          {/* Status Filter */}
          <div className="col-md-2">
            <select 
              className="form-select form-select-sm"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
            >
              <option value="ALL">All Statuses</option>
              {uniqueStatuses.map(s => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>
          
          {/* Type Filter */}
          <div className="col-md-2">
            <select 
              className="form-select form-select-sm"
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
            >
              <option value="ALL">All Types</option>
              {uniqueTypes.map(t => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>
          
          {/* Page Size */}
          <div className="col-md-2">
            <select 
              className="form-select form-select-sm"
              value={pageSize}
              onChange={(e) => setPageSize(Number(e.target.value))}
            >
              <option value={10}>10 per page</option>
              <option value={20}>20 per page</option>
              <option value={50}>50 per page</option>
              <option value={100}>100 per page</option>
            </select>
          </div>
          
          {/* Clear Filters */}
          <div className="col-md-2">
            {(statusFilter !== 'ALL' || typeFilter !== 'ALL' || searchQuery) && (
              <button 
                className="btn btn-sm btn-outline-warning w-100"
                onClick={() => {
                  setStatusFilter('ALL');
                  setTypeFilter('ALL');
                  setSearchQuery('');
                }}
              >
                Clear Filters
              </button>
            )}
          </div>
        </div>
      </div>
      
      <div className="card-body p-0">
        <div className="table-responsive">
          <table className="table table-hover mb-0">
            <thead className="table-light">
              <tr>
                <th 
                  style={{ cursor: 'pointer' }} 
                  onClick={() => handleSort('id')}
                >
                  ID <SortIcon field="id" />
                </th>
                <th 
                  style={{ cursor: 'pointer' }} 
                  onClick={() => handleSort('type')}
                >
                  Type <SortIcon field="type" />
                </th>
                <th 
                  style={{ cursor: 'pointer' }} 
                  onClick={() => handleSort('status')}
                >
                  Status <SortIcon field="status" />
                </th>
                <th 
                  style={{ cursor: 'pointer' }} 
                  onClick={() => handleSort('attempts')}
                >
                  Attempts <SortIcon field="attempts" />
                </th>
                <th 
                  style={{ cursor: 'pointer' }} 
                  onClick={() => handleSort('next_run_at')}
                >
                  Next Run <SortIcon field="next_run_at" />
                </th>
                <th>Details</th>
                <th>User</th>
              </tr>
            </thead>
            <tbody>
              {loading && tasks.length === 0 ? (
                <tr>
                  <td colSpan={7} className="text-center py-4 text-muted">
                    Loading...
                  </td>
                </tr>
              ) : paginatedTasks.length === 0 ? (
                <tr>
                  <td colSpan={7} className="text-center py-4 text-muted">
                    {tasks.length === 0 ? 'Queue is empty' : 'No matching tasks'}
                  </td>
                </tr>
              ) : (
                paginatedTasks.map((task) => {
                  const timeUntil = task.next_run_at ? getTimeUntil(task.next_run_at) : '';
                  const canExecute = task.status === 'PENDING' || task.status === 'FAILED';

                  return (
                    <tr key={task.id}>
                      <td>#{task.id}</td>
                      <td>
                        <span
                          className={`badge bg-info text-dark ${canExecute ? 'cursor-pointer' : ''}`}
                          onClick={() => canExecute && executeTask(task.id)}
                          style={{ cursor: canExecute ? 'pointer' : 'default' }}
                          title={canExecute ? 'Click to execute immediately' : ''}
                        >
                          {task.type}
                          {executing === task.id && (
                            <span
                              className="spinner-border spinner-border-sm ms-1"
                              style={{ width: '0.6em', height: '0.6em' }}
                            ></span>
                          )}
                          {canExecute && executing !== task.id && ' ▶'}
                        </span>
                      </td>
                      <td>
                        <span
                          className={`badge ${
                            task.status === 'PENDING'
                              ? 'bg-warning text-dark'
                              : task.status === 'PROCESSING'
                                ? 'bg-primary'
                                : task.status === 'COMPLETED'
                                  ? 'bg-success'
                                  : 'bg-danger'
                          }`}
                        >
                          {task.status}
                        </span>
                      </td>
                      <td>{task.attempts}</td>
                      <td>
                        <div>
                          {formatDate(task.next_run_at)}
                          {timeUntil && task.status === 'PENDING' && (
                            <div className="small text-muted">({timeUntil})</div>
                          )}
                        </div>
                      </td>
                      <td style={{ minWidth: '200px' }}>
                        {formatTaskDetails(task)}
                      </td>
                      <td className="small text-muted">
                        {task.username || '—'}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
      
      {/* Pagination */}
      {totalPages > 1 && (
        <div className="card-footer bg-light d-flex justify-content-between align-items-center">
          <div className="text-muted small">
            Showing {(page - 1) * pageSize + 1}-{Math.min(page * pageSize, filteredTasks.length)} of {filteredTasks.length}
          </div>
          <nav>
            <ul className="pagination pagination-sm mb-0">
              <li className={`page-item ${page === 1 ? 'disabled' : ''}`}>
                <button 
                  className="page-link" 
                  onClick={() => setPage(1)}
                  disabled={page === 1}
                >
                  «
                </button>
              </li>
              <li className={`page-item ${page === 1 ? 'disabled' : ''}`}>
                <button 
                  className="page-link" 
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  ‹
                </button>
              </li>
              
              {/* Page numbers */}
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                let pageNum: number;
                if (totalPages <= 5) {
                  pageNum = i + 1;
                } else if (page <= 3) {
                  pageNum = i + 1;
                } else if (page >= totalPages - 2) {
                  pageNum = totalPages - 4 + i;
                } else {
                  pageNum = page - 2 + i;
                }
                return (
                  <li key={pageNum} className={`page-item ${page === pageNum ? 'active' : ''}`}>
                    <button 
                      className="page-link" 
                      onClick={() => setPage(pageNum)}
                    >
                      {pageNum}
                    </button>
                  </li>
                );
              })}
              
              <li className={`page-item ${page === totalPages ? 'disabled' : ''}`}>
                <button 
                  className="page-link" 
                  onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                >
                  ›
                </button>
              </li>
              <li className={`page-item ${page === totalPages ? 'disabled' : ''}`}>
                <button 
                  className="page-link" 
                  onClick={() => setPage(totalPages)}
                  disabled={page === totalPages}
                >
                  »
                </button>
              </li>
            </ul>
          </nav>
        </div>
      )}
    </div>
  );
}
