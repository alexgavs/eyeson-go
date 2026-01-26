// Navbar Component with Theme Support
const API_BASE = '/api/v1';

// Theme Management - Initialize immediately (before DOM ready)
class ThemeManager {
    constructor() {
        this.theme = localStorage.getItem('theme') || 'dark';
        this.applyTheme();
    }

    applyTheme() {
        // Apply to <html> immediately
        document.documentElement.setAttribute('data-theme', this.theme);
        // Apply to body if exists
        if (document.body) {
            document.body.setAttribute('data-theme', this.theme);
        }
    }

    setTheme(theme) {
        this.theme = theme;
        localStorage.setItem('theme', this.theme);
        this.applyTheme();
        this.updateUI();
        return this.theme;
    }

    toggle() {
        return this.setTheme(this.theme === 'dark' ? 'light' : 'dark');
    }

    updateUI() {
        const themeIcon = document.getElementById('theme-icon');
        const themeLabel = document.getElementById('theme-label');
        if (themeIcon) {
            themeIcon.className = 'bi ' + this.getIcon();
        }
        if (themeLabel) {
            themeLabel.textContent = this.getLabel();
        }
    }

    getIcon() {
        return this.theme === 'dark' ? 'bi-sun' : 'bi-moon-stars';
    }

    getLabel() {
        return this.theme === 'dark' ? 'Light' : 'Dark';
    }

    getTheme() {
        return this.theme;
    }
}

// Create theme manager immediately - applies theme before page renders
const themeManager = new ThemeManager();

// Also apply when body loads (for pages that load navbar.js in head)
document.addEventListener('DOMContentLoaded', () => {
    themeManager.applyTheme();
});

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
                const payload = JSON.parse(atob(this.token.split('.')[1]));
                this.currentUser = users.find(u => u.id === payload.user_id) || users[0];
            }
        } catch (error) {
            console.error('Error loading user:', error);
        }
    }

    render() {
        const isAdmin = this.currentUser?.role_name === 'Administrator';
        const currentPath = window.location.pathname;

        this.container.innerHTML = `
            <nav class="navbar navbar-expand-lg">
                <div class="container-fluid">
                    <a class="navbar-brand fw-bold" href="/" style="color: white;">
                        <i class="bi bi-shield-check"></i> EyesOn Dashboard
                    </a>
                    <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav" style="border-color: rgba(255,255,255,0.3);">
                        <span class="navbar-toggler-icon"></span>
                    </button>
                    <div class="collapse navbar-collapse" id="navbarNav">
                        <ul class="navbar-nav me-auto">
                            <li class="nav-item">
                                <a class="nav-link ${currentPath === '/' || currentPath === '/index.html' || currentPath === '/dashboard.html' ? 'active fw-bold' : ''}" href="/" style="color: rgba(255,255,255,0.9);">
                                    <i class="bi bi-sim"></i> SIM Cards
                                </a>
                            </li>
                            <li class="nav-item">
                                <a class="nav-link ${currentPath === '/jobs.html' ? 'active fw-bold' : ''}" href="/jobs.html" style="color: rgba(255,255,255,0.9);">
                                    <i class="bi bi-clock-history"></i> Jobs
                                </a>
                            </li>
                            ${isAdmin ? `
                            <li class="nav-item">
                                <a class="nav-link ${currentPath === '/admin.html' ? 'active fw-bold' : ''}" href="/admin.html" style="color: rgba(255,255,255,0.9);">
                                    <i class="bi bi-shield-lock"></i> Admin
                                </a>
                            </li>
                            ` : ''}
                        </ul>
                        <ul class="navbar-nav align-items-center">
                            <li class="nav-item me-2">
                                <button class="theme-toggle" onclick="toggleTheme()" title="Toggle Theme">
                                    <i class="bi ${themeManager.getIcon()}" id="theme-icon"></i>
                                    <span id="theme-label">${themeManager.getLabel()}</span>
                                </button>
                            </li>
                            <li class="nav-item dropdown">
                                <a class="nav-link dropdown-toggle d-flex align-items-center" href="#" id="userDropdown" role="button" data-bs-toggle="dropdown" style="color: white;">
                                    <i class="bi bi-person-circle me-2"></i>
                                    <span>${this.currentUser?.username || 'User'}</span>
                                    <span class="badge ms-2" style="background: rgba(255,255,255,0.2); color: white;">${this.currentUser?.role_name || 'Role'}</span>
                                </a>
                                <ul class="dropdown-menu dropdown-menu-end">
                                    <li>
                                        <a class="dropdown-item" href="/profile.html">
                                            <i class="bi bi-person me-2"></i> My Profile
                                        </a>
                                    </li>
                                    ${isAdmin ? `
                                    <li><hr class="dropdown-divider"></li>
                                    <li>
                                        <a class="dropdown-item" href="/admin.html">
                                            <i class="bi bi-shield-lock me-2"></i> Admin Panel
                                        </a>
                                    </li>
                                    ` : ''}
                                    <li><hr class="dropdown-divider"></li>
                                    <li>
                                        <a class="dropdown-item" href="#" onclick="logout()" style="color: var(--danger);">
                                            <i class="bi bi-box-arrow-right me-2"></i> Logout
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
}

// Global functions
function toggleTheme() {
    themeManager.toggle();
}

// Allow setting specific theme
function setTheme(theme) {
    themeManager.setTheme(theme);
}

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
