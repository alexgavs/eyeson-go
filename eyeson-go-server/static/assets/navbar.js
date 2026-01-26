// Navbar Component
const API_BASE = '/api/v1';

class Navbar {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.token = localStorage.getItem('token');
        this.currentUser = null;
        this.init();
    }

    async init() {
        if (!this.token) {
            window.location.href = '/login.html';
            return;
        }

        await this.loadCurrentUser();
        this.render();
    }

    async loadCurrentUser() {
        try {
            const response = await fetch(`${API_BASE}/users`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });

            if (response.ok) {
                const users = await response.json();
                // Find current user by decoding JWT token
                const payload = JSON.parse(atob(this.token.split('.')[1]));
                this.currentUser = users.find(u => u.id === payload.user_id) || users[0];
            }
        } catch (error) {
            console.error('Error loading user:', error);
        }
    }

    render() {
        const isAdmin = this.currentUser?.role_name === 'Administrator';
        const isModerator = this.currentUser?.role_name === 'Moderator';

        this.container.innerHTML = `
            <nav class="navbar navbar-expand-lg navbar-dark" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
                <div class="container-fluid">
                    <a class="navbar-brand fw-bold" href="/">
                        🛡️ EyesOn Dashboard
                    </a>
                    <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
                        <span class="navbar-toggler-icon"></span>
                    </button>
                    <div class="collapse navbar-collapse" id="navbarNav">
                        <ul class="navbar-nav me-auto">
                            <li class="nav-item">
                                <a class="nav-link ${this.isActive('/') ? 'active' : ''}" href="/">
                                    <i class="bi bi-sim"></i> SIM Cards
                                </a>
                            </li>
                            <li class="nav-item">
                                <a class="nav-link ${this.isActive('/jobs.html') ? 'active' : ''}" href="/jobs.html">
                                    <i class="bi bi-clock-history"></i> Jobs
                                </a>
                            </li>
                            <li class="nav-item">
                                <a class="nav-link ${this.isActive('/stats.html') ? 'active' : ''}" href="/stats.html">
                                    <i class="bi bi-bar-chart"></i> Statistics
                                </a>
                            </li>
                            ${isAdmin ? `
                            <li class="nav-item">
                                <a class="nav-link ${this.isActive('/admin.html') ? 'active' : ''}" href="/admin.html">
                                    <i class="bi bi-shield-lock"></i> Admin Panel
                                </a>
                            </li>
                            ` : ''}
                        </ul>
                        <ul class="navbar-nav">
                            <li class="nav-item dropdown">
                                <a class="nav-link dropdown-toggle" href="#" id="userDropdown" role="button" data-bs-toggle="dropdown">
                                    <i class="bi bi-person-circle"></i> ${this.currentUser?.username || 'User'}
                                    <span class="badge bg-light text-dark ms-2">${this.currentUser?.role_name || 'Role'}</span>
                                </a>
                                <ul class="dropdown-menu dropdown-menu-end" style="background: #253346; border-color: #4a5568;">
                                    <li>
                                        <a class="dropdown-item" href="/profile.html" style="color: #e0e0e0;">
                                            <i class="bi bi-person"></i> My Profile
                                        </a>
                                    </li>
                                    ${isAdmin ? `
                                    <li><hr class="dropdown-divider" style="border-color: #4a5568;"></li>
                                    <li>
                                        <a class="dropdown-item" href="/admin.html" style="color: #e0e0e0;">
                                            <i class="bi bi-shield-lock"></i> Admin Panel
                                        </a>
                                    </li>
                                    ` : ''}
                                    <li><hr class="dropdown-divider" style="border-color: #4a5568;"></li>
                                    <li>
                                        <a class="dropdown-item" href="#" onclick="logout()" style="color: #dc3545;">
                                            <i class="bi bi-box-arrow-right"></i> Logout
                                        </a>
                                    </li>
                                </ul>
                            </li>
                        </ul>
                    </div>
                </div>
            </nav>
        `;
    }

    isActive(path) {
        return window.location.pathname === path;
    }
}

// Global logout function
function logout() {
    localStorage.removeItem('token');
    window.location.href = '/login.html';
}

// Auto-initialize navbar if element exists
document.addEventListener('DOMContentLoaded', () => {
    const navbarEl = document.getElementById('navbar');
    if (navbarEl) {
        new Navbar('navbar');
    }
});
