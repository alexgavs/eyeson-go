// Admin Panel JavaScript
const API_BASE = '/api/v1';
let token = localStorage.getItem('token');
let currentUser = null;
let users = [];
let roles = [];

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    if (!token) {
        window.location.href = '/login.html';
        return;
    }

    loadUsers();
    loadRoles();
    getCurrentUser();
});

// Get current user info
async function getCurrentUser() {
    try {
        const response = await fetch(`${API_BASE}/users`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.ok) {
            const data = await response.json();
            if (data.length > 0) {
                currentUser = data[0];
                document.getElementById('current-user').textContent = currentUser.username;
            }
        }
    } catch (error) {
        console.error('Error fetching current user:', error);
    }
}

// Tab switching
function showTab(tabName) {
    document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));

    event.target.classList.add('active');
    document.getElementById(`${tabName}-tab`).classList.add('active');

    if (tabName === 'users') {
        loadUsers();
    } else if (tabName === 'roles') {
        loadRoles();
    }
}

// Users Management
async function loadUsers() {
    const loading = document.getElementById('users-loading');
    const error = document.getElementById('users-error');
    const table = document.getElementById('users-table');
    const tbody = document.getElementById('users-tbody');

    loading.style.display = 'block';
    error.style.display = 'none';
    table.style.display = 'none';

    try {
        const response = await fetch(`${API_BASE}/users`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (!response.ok) {
            throw new Error('Failed to load users');
        }

        users = await response.json();

        tbody.innerHTML = users.map(user => `
            <tr>
                <td>${user.id}</td>
                <td><strong>${user.username}</strong></td>
                <td><span class="badge badge-primary">${user.role_name || 'No Role'}</span></td>
                <td>
                    ${user.is_active
                        ? '<span class="badge badge-success">Active</span>'
                        : '<span class="badge badge-danger">Inactive</span>'}
                </td>
                <td>${formatDate(user.last_seen)}</td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-primary btn-sm" onclick="editUser(${user.id})">Edit</button>
                        <button class="btn btn-primary btn-sm" onclick="resetPassword(${user.id})">Reset Password</button>
                        <button class="btn btn-danger btn-sm" onclick="deleteUser(${user.id}, '${user.username}')">Delete</button>
                    </div>
                </td>
            </tr>
        `).join('');

        loading.style.display = 'none';
        table.style.display = 'table';
    } catch (err) {
        loading.style.display = 'none';
        error.textContent = err.message;
        error.style.display = 'block';
    }
}

function showCreateUserModal() {
    document.getElementById('user-modal-title').textContent = 'Create User';
    document.getElementById('user-form').reset();
    document.getElementById('user-id').value = '';
    document.getElementById('password-group').style.display = 'block';
    document.getElementById('user-password').required = true;

    // Load roles into select
    loadRolesIntoSelect();

    document.getElementById('user-modal').classList.add('active');
}

async function loadRolesIntoSelect() {
    const select = document.getElementById('user-role');
    select.innerHTML = '<option value="">Select role...</option>';

    try {
        const response = await fetch(`${API_BASE}/roles`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.ok) {
            const roles = await response.json();
            roles.forEach(role => {
                const option = document.createElement('option');
                option.value = role.ID;
                option.textContent = role.Name;
                select.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading roles:', error);
    }
}

function editUser(userId) {
    const user = users.find(u => u.id === userId);
    if (!user) return;

    document.getElementById('user-modal-title').textContent = 'Edit User';
    document.getElementById('user-id').value = user.id;
    document.getElementById('user-username').value = user.username;
    document.getElementById('user-role').value = user.role_id;
    document.getElementById('user-active').checked = user.is_active;
    document.getElementById('password-group').style.display = 'none';
    document.getElementById('user-password').required = false;

    loadRolesIntoSelect();

    document.getElementById('user-modal').classList.add('active');
}

async function saveUser(event) {
    event.preventDefault();

    const userId = document.getElementById('user-id').value;
    const username = document.getElementById('user-username').value;
    const password = document.getElementById('user-password').value;
    const roleId = parseInt(document.getElementById('user-role').value);
    const isActive = document.getElementById('user-active').checked;

    try {
        let response;

        if (userId) {
            // Update existing user
            response = await fetch(`${API_BASE}/users/${userId}`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, role_id: roleId, is_active: isActive })
            });
        } else {
            // Create new user
            if (!password) {
                alert('Password is required for new users');
                return;
            }

            response = await fetch(`${API_BASE}/users`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password, role_id: roleId })
            });
        }

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save user');
        }

        closeModal('user-modal');
        loadUsers();
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

async function resetPassword(userId) {
    const newPassword = prompt('Enter new password (min 6 characters):');
    if (!newPassword || newPassword.length < 6) {
        alert('Password must be at least 6 characters');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/users/${userId}/reset-password`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ new_password: newPassword })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to reset password');
        }

        alert('Password reset successfully');
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

async function deleteUser(userId, username) {
    if (!confirm(`Are you sure you want to delete user "${username}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/users/${userId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete user');
        }

        loadUsers();
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

// Roles Management
async function loadRoles() {
    const loading = document.getElementById('roles-loading');
    const error = document.getElementById('roles-error');
    const table = document.getElementById('roles-table');
    const tbody = document.getElementById('roles-tbody');

    loading.style.display = 'block';
    error.style.display = 'none';
    table.style.display = 'none';

    try {
        const response = await fetch(`${API_BASE}/roles`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (!response.ok) {
            throw new Error('Failed to load roles');
        }

        roles = await response.json();

        tbody.innerHTML = roles.map(role => `
            <tr>
                <td>${role.ID}</td>
                <td><strong>${role.Name}</strong></td>
                <td>${role.Description}</td>
                <td>
                    <div class="permissions-list">
                        ${role.Permissions.split(',').map(p =>
                            `<span class="badge badge-primary">${p.trim()}</span>`
                        ).join('')}
                    </div>
                </td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-primary btn-sm" onclick="editRole(${role.ID})">Edit</button>
                        <button class="btn btn-danger btn-sm" onclick="deleteRole(${role.ID}, '${role.Name}')">Delete</button>
                    </div>
                </td>
            </tr>
        `).join('');

        loading.style.display = 'none';
        table.style.display = 'table';
    } catch (err) {
        loading.style.display = 'none';
        error.textContent = err.message;
        error.style.display = 'block';
    }
}

function showCreateRoleModal() {
    document.getElementById('role-modal-title').textContent = 'Create Role';
    document.getElementById('role-form').reset();
    document.getElementById('role-id').value = '';
    document.getElementById('role-modal').classList.add('active');
}

function editRole(roleId) {
    const role = roles.find(r => r.ID === roleId);
    if (!role) return;

    document.getElementById('role-modal-title').textContent = 'Edit Role';
    document.getElementById('role-id').value = role.ID;
    document.getElementById('role-name').value = role.Name;
    document.getElementById('role-description').value = role.Description;
    document.getElementById('role-permissions').value = role.Permissions;

    document.getElementById('role-modal').classList.add('active');
}

async function saveRole(event) {
    event.preventDefault();

    const roleId = document.getElementById('role-id').value;
    const name = document.getElementById('role-name').value;
    const description = document.getElementById('role-description').value;
    const permissions = document.getElementById('role-permissions').value;

    try {
        let response;

        if (roleId) {
            // Update existing role
            response = await fetch(`${API_BASE}/roles/${roleId}`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name, description, permissions })
            });
        } else {
            // Create new role
            response = await fetch(`${API_BASE}/roles`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name, description, permissions })
            });
        }

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save role');
        }

        closeModal('role-modal');
        loadRoles();
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

async function deleteRole(roleId, roleName) {
    if (!confirm(`Are you sure you want to delete role "${roleName}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/roles/${roleId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete role');
        }

        loadRoles();
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

// Utility functions
function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

function formatDate(dateString) {
    if (!dateString || dateString === '0001-01-01T00:00:00Z') {
        return 'Never';
    }
    const date = new Date(dateString);
    return date.toLocaleString();
}

function logout() {
    localStorage.removeItem('token');
    window.location.href = '/login.html';
}

// Close modal on background click
document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.classList.remove('active');
        }
    });
});
