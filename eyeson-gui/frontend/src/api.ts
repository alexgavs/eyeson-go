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

export const ChangeStatus = async (msisdns: string[], status: string) => {
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
        
        if (response.ok) return "SUCCESS";
        
        // Более понятное сообщение об ошибке
        let errorMsg = data.error || "Status change failed";
        if (errorMsg.includes("Permission Denied")) {
            errorMsg = "Permission Denied: У API пользователя нет прав на изменение этой SIM-карты";
        }
        return errorMsg;
    } catch (e) {
        console.error('[ChangeStatus] Error:', e);
        return "Network Error: " + e;
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
