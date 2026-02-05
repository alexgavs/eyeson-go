import React, { useEffect, useState } from 'react';
import { GetSimHistory, type SimHistory } from '../api';
import { StatusBadge } from './StatusBadges';

export interface SimDetailModalProps {
  sim: any;
  editMode: boolean;
  editValues: any;
  loading: boolean;
  onClose: () => void;
  onEditStart: () => void;
  onEditSave: () => void;
  onEditValueChange: (key: string, value: string) => void;
  onStatusChange: (status: string) => void;
}

export function SimDetailModal({
  sim,
  editMode,
  editValues,
  loading,
  onClose,
  onEditStart,
  onEditSave,
  onEditValueChange,
  onStatusChange
}: SimDetailModalProps) {
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const [activeTab, setActiveTab] = useState<'details' | 'history'>('details');
  const [history, setHistory] = useState<SimHistory[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);

  const modalRef = React.useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (activeTab === 'history' && sim.MSISDN) {
      setHistoryLoading(true);
      GetSimHistory(sim.MSISDN)
        .then((data) => setHistory(data))
        .finally(() => setHistoryLoading(false));
    }
  }, [activeTab, sim.MSISDN]);

  const handleMouseDown = (e: React.MouseEvent) => {
    if ((e.target as HTMLElement).closest('.modal-header')) {
      setIsDragging(true);
      setDragStart({ x: e.clientX - position.x, y: e.clientY - position.y });
      e.preventDefault();
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      setPosition({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y
      });
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  return (
    <div
      className="modal fade show d-block"
      style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}
      tabIndex={-1}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
    >
      <div
        ref={modalRef}
        className="modal-dialog modal-lg"
        style={{
          transform: `translate(${position.x}px, ${position.y}px)`,
          margin: '1.75rem auto',
          transition: isDragging ? 'none' : 'transform 0.1s ease-out'
        }}
        onMouseDown={handleMouseDown}
      >
        <div className="modal-content bg-dark border-secondary text-light">
          <div className="modal-header border-secondary" style={{ cursor: 'move', userSelect: 'none' }}>
            <h5 className="modal-title">📱 SIM Details: {sim.MSISDN}</h5>
            <button type="button" className="btn-close btn-close-white" onClick={onClose}></button>
          </div>
          <div className="modal-body">
            <ul className="nav nav-tabs border-secondary mb-3">
              <li className="nav-item">
                <button
                  className={`nav-link ${
                    activeTab === 'details'
                      ? 'active bg-dark text-white border-secondary border-bottom-0'
                      : 'text-muted'
                  }`}
                  onClick={() => setActiveTab('details')}
                >
                  📋 Details
                </button>
              </li>
              <li className="nav-item">
                <button
                  className={`nav-link ${
                    activeTab === 'history'
                      ? 'active bg-dark text-white border-secondary border-bottom-0'
                      : 'text-muted'
                  }`}
                  onClick={() => setActiveTab('history')}
                >
                  📜 Status History
                </button>
              </li>
            </ul>

            {activeTab === 'details' ? (
              editMode ? (
                <div className="row g-3">
                  <div className="col-12">
                    <h6 className="text-primary">Edit Labels</h6>
                  </div>
                  <div className="col-md-12">
                    <label className="form-label text-muted small">Label 1 (SIM Label)</label>
                    <input
                      type="text"
                      className="form-control bg-dark text-light border-secondary"
                      value={editValues.label1}
                      onChange={(e) => onEditValueChange('label1', e.target.value)}
                    />
                  </div>
                  <div className="col-md-12">
                    <label className="form-label text-muted small">Label 2 (Group Tag)</label>
                    <input
                      type="text"
                      className="form-control bg-dark text-light border-secondary"
                      value={editValues.label2}
                      onChange={(e) => onEditValueChange('label2', e.target.value)}
                    />
                  </div>
                  <div className="col-md-12">
                    <label className="form-label text-muted small">Label 3 (Device Tag)</label>
                    <input
                      type="text"
                      className="form-control bg-dark text-light border-secondary"
                      value={editValues.label3}
                      onChange={(e) => onEditValueChange('label3', e.target.value)}
                    />
                  </div>
                </div>
              ) : (
                <div className="row g-3">
                  <div className="col-md-6">
                    <label className="text-muted small">CLI (Local Number)</label>
                    <div className="fw-bold">{sim.CLI}</div>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">ICCID</label>
                    <div className="fw-bold text-break">{sim.SIM_SWAP}</div>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">Status</label>
                    <div>
                      <StatusBadge
                        status={sim.SIM_STATUS_CHANGE}
                        isPending={sim._pending}
                        syncStatus={sim.SYNC_STATUS}
                      />
                    </div>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">IMSI</label>
                    <div className="fw-bold">{sim.IMSI}</div>
                  </div>
                  <div className="col-12">
                    <hr className="border-secondary" />
                    <h6 className="text-primary mb-3">Plan & Usage</h6>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">Rate Plan</label>
                    <div>{sim.RATE_PLAN_FULL_NAME}</div>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">APN</label>
                    <div>{sim.APN_NAME}</div>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">IP Address</label>
                    <div>{sim.IP1 || '-'}</div>
                  </div>
                  <div className="col-md-6">
                    <label className="text-muted small">Monthly Usage</label>
                    <div>
                      {sim.MONTHLY_USAGE_MB} MB / {sim.ALLOCATED_MB} MB
                    </div>
                  </div>
                  <div className="col-12">
                    <hr className="border-secondary" />
                    <h6 className="text-primary mb-3">Labels</h6>
                  </div>
                  <div className="col-md-4">
                    <label className="text-muted small">Label 1</label>
                    <div>{sim.CUSTOMER_LABEL_1 || '-'}</div>
                  </div>
                  <div className="col-md-4">
                    <label className="text-muted small">Label 2</label>
                    <div>{sim.CUSTOMER_LABEL_2 || '-'}</div>
                  </div>
                  <div className="col-md-4">
                    <label className="text-muted small">Label 3</label>
                    <div>{sim.CUSTOMER_LABEL_3 || '-'}</div>
                  </div>
                </div>
              )
            ) : (
              <div className="table-responsive">
                {historyLoading ? (
                  <div className="text-center py-4">
                    <div className="spinner-border text-primary" role="status"></div>
                  </div>
                ) : (
                  <table className="table table-dark table-sm table-striped">
                    <thead>
                      <tr>
                        <th>Time</th>
                        <th>Action</th>
                        <th>Field</th>
                        <th>Change</th>
                        <th>Source</th>
                      </tr>
                    </thead>
                    <tbody>
                      {history.length > 0 ? (
                        history.map((h) => (
                          <tr key={h.id}>
                            <td>
                              <small>{new Date(h.created_at).toLocaleString()}</small>
                            </td>
                            <td>{h.action}</td>
                            <td>{h.field}</td>
                            <td>
                              <small className="text-muted">{h.old_value}</small>
                              <span className="mx-1">→</span>
                              <span className="text-info">{h.new_value}</span>
                            </td>
                            <td>
                              <span className="badge bg-secondary">{h.source}</span>
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={5} className="text-center text-muted">
                            No history records found
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                )}
              </div>
            )}
          </div>
          <div className="modal-footer justify-content-between border-secondary">
            <div>
              {!editMode && activeTab === 'details' && (
                <div className="btn-group">
                  <button
                    className="btn btn-outline-success btn-sm"
                    onClick={() => onStatusChange('Activated')}
                    disabled={sim._pending}
                  >
                    Activate
                  </button>
                  <button
                    className="btn btn-outline-warning btn-sm"
                    onClick={() => onStatusChange('Suspended')}
                    disabled={sim._pending}
                  >
                    Suspend
                  </button>
                  <button
                    className="btn btn-outline-danger btn-sm"
                    onClick={() => onStatusChange('Terminated')}
                    disabled={sim._pending}
                  >
                    Terminate
                  </button>
                </div>
              )}
            </div>
            <div className="d-flex gap-2">
              <button type="button" className="btn btn-secondary" onClick={onClose}>
                Close
              </button>
              {editMode && activeTab === 'details' && (
                <button type="button" className="btn btn-success" onClick={onEditSave} disabled={loading}>
                  {loading ? 'Saving...' : 'Save Changes'}
                </button>
              )}
              {!editMode && activeTab === 'details' && (
                <button type="button" className="btn btn-primary" onClick={onEditStart}>
                  Edit Labels
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
