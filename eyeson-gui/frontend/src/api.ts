// API Service replacement for Wails bindings

const BASE_URL = "/api/v1";

export const Login = async (username: string, password: string) => {
    try {
        const response = await fetch(`${BASE_URL}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        
        const data = await response.json();
        
        if (response.ok && data.token) {
            // Save token
            localStorage.setItem('token', data.token);
            return "SUCCESS";
        }
        return data.error || "Login failed";
    } catch (e) {
        return "Network Error: " + e;
    }
};

export const GetStats = async (forceRefresh: boolean = false) => {
    try {
        const token = localStorage.getItem('token');
        const url = forceRefresh ? `${BASE_URL}/stats?forceRefresh=true` : `${BASE_URL}/stats`;
        const response = await fetch(url, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        const data = await response.json();
        if (response.ok) return data.data;
        return null;
    } catch (e) {
        console.error(e);
        return null;
    }
};

export const GetSims = async (search: string, start: number = 0, limit: number = 25, sortBy: string = "", sortDirection: string = "ASC", status: string = "") => {
    try {
        const token = localStorage.getItem('token');
        const params = new URLSearchParams();
        if (search) params.append('search', search);
        if (status) params.append('status', status);
        
        params.append('start', start.toString());
        params.append('limit', limit.toString());
        if (sortBy) {
            params.append('sortBy', sortBy);
            params.append('sortDirection', sortDirection);
        }

        const response = await fetch(`${BASE_URL}/sims?${params}`, {
            headers: {
                'Authorization': `Bearer ${token}` 
            }
        });
        
        const data = await response.json();
        if (response.ok) {
            return data;
        }
        throw new Error(data.error || "Failed to fetch sims");
    } catch (e) {
        console.error(e);
        return { data: [], count: 0 };
    }
};

export const UpdateSim = async (msisdn: string, field: string, value: string) => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/sims/update`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ msisdn, field, value })
        });
        
        const data = await response.json();
        if (response.ok) return "SUCCESS";
        return data.error || "Update failed";
    } catch (e) {
        return "Network Error: " + e;
    }
};

export interface ChangeStatusResult {
    success: boolean;
    requestId?: number;
    error?: string;
}

export const ChangeStatus = async (msisdns: string[], status: string): Promise<ChangeStatusResult> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/sims/bulk-status`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ status, msisdns })
        });
        
        const data = await response.json();
        console.log('[ChangeStatus] Response:', response.status, data);
        
        if (response.ok && (data.result === "SUCCESS" || data.result === "queued")) {
            return { success: true, requestId: data.requestId };
        }
        
        // Более понятное сообщение об ошибке
        let errorMsg = data.error || data.message || "Status change failed";
        if (errorMsg.includes("Permission Denied")) {
            errorMsg = "Permission Denied: У API пользователя нет прав на изменение этой SIM-карты";
        }
        return { success: false, error: errorMsg };
    } catch (e) {
        console.error('[ChangeStatus] Error:', e);
        return { success: false, error: "Network Error: " + e };
    }
};

// Получить статус Job по ID (для polling)
export const GetJobStatus = async (jobId: number) => {
    try {
        const token = localStorage.getItem('token');
        // Use local endpoint since we are tracking internal SyncTasks
        const response = await fetch(`${BASE_URL}/jobs/local/${jobId}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();
        if (response.ok) {
            return data; // Returns { jobStatus: "...", ... }
        }
        return null;
    } catch (e) {
        console.error('[GetJobStatus] Error:', e);
        return null;
    }
};

export const GetJobs = async (page: number = 1, perPage: number = 25, jobId?: number, status?: string) => {
    try {
        const token = localStorage.getItem('token');
        const params = new URLSearchParams();
        params.append('page', page.toString());
        params.append('per_page', perPage.toString());
        if (jobId) params.append('job_id', jobId.toString());
        if (status) params.append('status', status);

        const response = await fetch(`${BASE_URL}/jobs?${params}`, {
            headers: {
                'Authorization': `Bearer ${token}` 
            }
        });
        
        const data = await response.json();
        if (response.ok) return data;
        throw new Error(data.error || "Failed to fetch jobs");
    } catch (e) {
        console.error(e);
        return { success: false, data: [], pagination: { page: 1, total: 0, total_pages: 1 } };
    }
};

// ==================== USER MANAGEMENT API ====================

export interface User {
    id: number;
    username: string;
    email: string;
    role: string;
    is_active: boolean;
    created_at: string;
    updated_at: string;
}

export interface Role {
    id: number;
    name: string;
    description: string;
    permissions: string[];
}

export const GetUsers = async (): Promise<User[]> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/users`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        if (response.ok) return data.data || [];
        throw new Error(data.error || "Failed to fetch users");
    } catch (e) {
        console.error(e);
        return [];
    }
};

export const CreateUser = async (user: { username: string; email: string; password: string; role: string }): Promise<{ success: boolean; error?: string }> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/users`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify(user)
        });
        
        const data = await response.json();
        if (response.ok) return { success: true };
        return { success: false, error: data.error || "Failed to create user" };
    } catch (e) {
        return { success: false, error: "Network Error: " + e };
    }
};

export const UpdateUser = async (id: number, user: { username?: string; email?: string; role?: string; is_active?: boolean }): Promise<{ success: boolean; error?: string }> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/users/${id}`, {
            method: 'PUT',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify(user)
        });
        
        const data = await response.json();
        if (response.ok) return { success: true };
        return { success: false, error: data.error || "Failed to update user" };
    } catch (e) {
        return { success: false, error: "Network Error: " + e };
    }
};

export const DeleteUser = async (id: number): Promise<{ success: boolean; error?: string }> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/users/${id}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        if (response.ok) return { success: true };
        return { success: false, error: data.error || "Failed to delete user" };
    } catch (e) {
        return { success: false, error: "Network Error: " + e };
    }
};

export const ResetUserPassword = async (id: number, newPassword: string): Promise<{ success: boolean; error?: string }> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/users/${id}/reset-password`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ new_password: newPassword })
        });
        
        const data = await response.json();
        if (response.ok) return { success: true };
        return { success: false, error: data.error || "Failed to reset password" };
    } catch (e) {
        return { success: false, error: "Network Error: " + e };
    }
};

export const GetRoles = async (): Promise<Role[]> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/roles`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        if (response.ok) return data.data || [];
        throw new Error(data.error || "Failed to fetch roles");
    } catch (e) {
        console.error(e);
        return [];
    }
};

// API Status types
export interface APIConnectionInfo {
    status: string;
    response_time_ms: number;
    details?: Record<string, string>;
    error?: string;
}

export interface APIStatusResponse {
    eyeson_api: APIConnectionInfo;
    go_backend: APIConnectionInfo;
    database: APIConnectionInfo;
    last_checked: string;
}

// Get API Status (Admin only)
export const GetAPIStatus = async (): Promise<APIStatusResponse | null> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/api-status`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (response.ok) {
            return await response.json();
        }
        return null;
    } catch (e) {
        console.error('Error fetching API status:', e);
        return null;
    }
};

// Queue Type Definitions
export interface QueueTask {
    id: number;
    type: string;
    payload: any;
    status: string;
    created_at: string;
    attempts: number;
    last_error: string;
}

// Get Queue
export const GetSyncQueue = async (): Promise<{count: number, data: QueueTask[]}> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/jobs/queue`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (response.ok) {
            return await response.json();
        }
        
        throw new Error("Failed to fetch queue");
    } catch (e) {
        console.error('Error fetching queue:', e);
        return { count: 0, data: [] };
    }
};

// Toggle API Connection (Simulate Network Failure or Set Mode)
export const ToggleAPIConnection = async (action: string, mode?: string): Promise<{result: string, message: string}> => {
    try {
        const token = localStorage.getItem('token');
        const body: any = { action };
        if (mode) body.mode = mode;

        const response = await fetch(`${BASE_URL}/api-connection`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify(body)
        });
        
        const data = await response.json();
        if (response.ok) return data;
        throw new Error(data.error || "Action failed");
    } catch (e) {
        console.error('Error toggling API connection:', e);
        throw e;
    }
};

// Execute Queue Task Immediately
export const ExecuteQueueTask = async (taskId: number): Promise<{result: string, message: string}> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/jobs/queue/${taskId}/execute`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        if (response.ok) return data;
        throw new Error(data.error || "Failed to execute task");
    } catch (e) {
        console.error('Error executing queue task:', e);
        throw e;
    }
};

export interface SimHistory {
    id: number;
    created_at: string;
    msisdn: string;
    action: string;
    field: string;
    old_value: string;
    new_value: string;
    source: string;
}

export const GetSimHistory = async (msisdn: string): Promise<SimHistory[]> => {
    try {
        const token = localStorage.getItem('token');
        const response = await fetch(`${BASE_URL}/sims/${msisdn}/history`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        if (response.ok) return data.data || [];
        throw new Error(data.error || "Failed to fetch history");
    } catch (e) {
        console.error(e);
        return [];
    }
};