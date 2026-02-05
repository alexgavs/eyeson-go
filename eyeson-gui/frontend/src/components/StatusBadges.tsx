import React from 'react';

export function StatusBadge({
  status,
  isPending = false,
  syncStatus
}: {
  status: string;
  isPending?: boolean;
  syncStatus?: string;
}) {
  let className = 'badge ';
  if (status === 'Activated') className += 'bg-success';
  else if (status === 'Suspended') className += 'bg-warning text-dark';
  else if (status === 'Terminated') className += 'bg-danger';
  else className += 'bg-secondary';

  const hasSyncStatus = !!syncStatus;
  const isQueueLike = syncStatus === 'PENDING' || syncStatus === 'PROCESSING' || syncStatus === 'IN_PROGRESS';
  const showSpinner = isPending || isQueueLike;

  return (
    <span className={className}>
      {status}
      {(showSpinner || hasSyncStatus) && (
        <>
          {showSpinner && (
            <span
              className="spinner-border spinner-border-sm ms-1"
              style={{ width: '0.7em', height: '0.7em' }}
            ></span>
          )}
          {hasSyncStatus && (
            <span className="ms-1 small fst-italic">
              {' '}
              {isQueueLike ? '(In Queue)' : `(${syncStatus})`}
            </span>
          )}
        </>
      )}
    </span>
  );
}

export function JobStatusBadge({ status }: { status: string }) {
  const badges: Record<string, string> = {
    PENDING: 'bg-secondary',
    IN_PROGRESS: 'bg-info',
    COMPLETED: 'bg-success',
    SUCCESS: 'bg-success',
    PARTIAL_SUCCESS: 'bg-warning',
    FAILED: 'bg-danger'
  };

  return <span className={`badge ${badges[status] || 'bg-secondary'}`}>{status}</span>;
}
