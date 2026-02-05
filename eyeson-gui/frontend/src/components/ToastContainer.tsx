import React from 'react';
import type { Toast } from '../types/app';

export function ToastContainer({ toasts }: { toasts: Toast[] }) {
  return (
    <div className="toast-container position-fixed bottom-0 end-0 p-3" style={{ zIndex: 9999 }}>
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className="toast show"
          role="alert"
          style={{
            backgroundColor:
              toast.type === 'danger'
                ? '#dc3545'
                : toast.type === 'success'
                  ? '#198754'
                  : toast.type === 'warning'
                    ? '#ffc107'
                    : '#0dcaf0',
            color: toast.type === 'warning' ? '#000' : '#fff'
          }}
        >
          <div className="toast-body">{toast.message}</div>
        </div>
      ))}
    </div>
  );
}
