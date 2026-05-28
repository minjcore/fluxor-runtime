// GoProxy Dashboard JavaScript

const API_BASE = '/api/v1';

// State
let currentModules = [];
let currentQuery = '';

// DOM Elements
const searchInput = document.getElementById('searchInput');
const searchBtn = document.getElementById('searchBtn');
const moduleList = document.getElementById('moduleList');
const sectionTitle = document.getElementById('sectionTitle');
const modalOverlay = document.getElementById('modalOverlay');
const modalTitle = document.getElementById('modalTitle');
const modalBody = document.getElementById('modalBody');
const modalClose = document.getElementById('modalClose');

// Stats elements
const totalModulesEl = document.getElementById('totalModules');
const totalVersionsEl = document.getElementById('totalVersions');
const totalDownloadsEl = document.getElementById('totalDownloads');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadStats();
    loadModules();
    setupEventListeners();
});

// Event Listeners
function setupEventListeners() {
    searchBtn.addEventListener('click', handleSearch);
    searchInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') handleSearch();
    });
    
    modalClose.addEventListener('click', closeModal);
    modalOverlay.addEventListener('click', (e) => {
        if (e.target === modalOverlay) closeModal();
    });
    
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') closeModal();
    });
}

// Load overall stats
async function loadStats() {
    try {
        const response = await fetch(`${API_BASE}/stats`);
        if (!response.ok) throw new Error('Failed to load stats');
        
        const data = await response.json();
        totalModulesEl.textContent = formatNumber(data.totalModules || 0);
        totalVersionsEl.textContent = formatNumber(data.totalVersions || 0);
        totalDownloadsEl.textContent = formatNumber(data.totalDownloads || 0);
    } catch (error) {
        console.error('Failed to load stats:', error);
        totalModulesEl.textContent = '-';
        totalVersionsEl.textContent = '-';
        totalDownloadsEl.textContent = '-';
    }
}

// Load modules list
async function loadModules(prefix = '') {
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/modules?prefix=${encodeURIComponent(prefix)}&limit=50`);
        if (!response.ok) throw new Error('Failed to load modules');
        
        const data = await response.json();
        currentModules = data.modules || [];
        sectionTitle.textContent = 'All Modules';
        renderModules(currentModules);
    } catch (error) {
        console.error('Failed to load modules:', error);
        showError('Failed to load modules. Please try again.');
    }
}

// Search modules
async function handleSearch() {
    const query = searchInput.value.trim();
    if (!query) {
        loadModules();
        return;
    }
    
    currentQuery = query;
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/modules/search?q=${encodeURIComponent(query)}&limit=50`);
        if (!response.ok) throw new Error('Failed to search modules');
        
        const data = await response.json();
        currentModules = data.results || [];
        sectionTitle.textContent = `Search Results for "${query}"`;
        renderModules(currentModules);
    } catch (error) {
        console.error('Search failed:', error);
        showError('Search failed. Please try again.');
    }
}

// Render modules list
function renderModules(modules) {
    if (!modules || modules.length === 0) {
        moduleList.innerHTML = `
            <div class="empty-state">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                    <line x1="9" y1="14" x2="15" y2="14"/>
                </svg>
                <h3>No modules found</h3>
                <p>Try a different search query or upload a new module.</p>
            </div>
        `;
        return;
    }
    
    moduleList.innerHTML = modules.map(module => {
        const path = typeof module === 'string' ? module : module.path;
        const version = module.latestVersion || 'N/A';
        const downloads = module.downloads || 0;
        const updatedAt = module.updatedAt ? formatDate(module.updatedAt) : 'N/A';
        
        return `
            <div class="module-card" onclick="openModuleDetail('${escapeHtml(path)}')">
                <div class="module-header">
                    <span class="module-path">${escapeHtml(path)}</span>
                    <span class="module-version">${escapeHtml(version)}</span>
                </div>
                <div class="module-meta">
                    <span>
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                            <polyline points="7 10 12 15 17 10"/>
                            <line x1="12" y1="15" x2="12" y2="3"/>
                        </svg>
                        ${formatNumber(downloads)} downloads
                    </span>
                    <span>
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <polyline points="12 6 12 12 16 14"/>
                        </svg>
                        Updated ${updatedAt}
                    </span>
                </div>
            </div>
        `;
    }).join('');
}

// Open module detail modal
async function openModuleDetail(modulePath) {
    modalTitle.textContent = modulePath;
    modalBody.innerHTML = '<div class="loading"><div class="spinner"></div><p>Loading...</p></div>';
    modalOverlay.classList.add('active');
    
    try {
        const response = await fetch(`${API_BASE}/modules/${encodeURIComponent(modulePath)}`);
        if (!response.ok) throw new Error('Failed to load module details');
        
        const data = await response.json();
        renderModuleDetail(data);
    } catch (error) {
        console.error('Failed to load module details:', error);
        modalBody.innerHTML = `
            <div class="empty-state">
                <p>Failed to load module details. Please try again.</p>
            </div>
        `;
    }
}

// Render module detail
function renderModuleDetail(module) {
    const versions = module.versions || [];
    const installCmd = `go get ${module.path}@latest`;
    
    modalBody.innerHTML = `
        <div class="install-cmd">
            <code id="installCmd">${escapeHtml(installCmd)}</code>
            <button class="copy-btn" onclick="copyToClipboard('${escapeHtml(installCmd)}', this)">
                Copy
            </button>
        </div>
        
        <div class="version-list">
            <h4 style="margin-bottom: 12px; color: var(--text-secondary);">
                Available Versions (${versions.length})
            </h4>
            ${versions.length > 0 ? versions.map(version => `
                <div class="version-item">
                    <span class="version-tag">${escapeHtml(version)}</span>
                    <button class="copy-btn" onclick="copyToClipboard('go get ${escapeHtml(module.path)}@${escapeHtml(version)}', this)">
                        Copy
                    </button>
                </div>
            `).join('') : '<p style="color: var(--text-secondary);">No versions available</p>'}
        </div>
        
        <div style="margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--border);">
            <h4 style="margin-bottom: 12px; color: var(--text-secondary);">GOPROXY Configuration</h4>
            <div class="install-cmd">
                <code>GOPROXY=${window.location.origin},direct</code>
                <button class="copy-btn" onclick="copyToClipboard('GOPROXY=${window.location.origin},direct', this)">
                    Copy
                </button>
            </div>
        </div>
    `;
}

// Close modal
function closeModal() {
    modalOverlay.classList.remove('active');
}

// Copy to clipboard
async function copyToClipboard(text, button) {
    try {
        await navigator.clipboard.writeText(text);
        button.textContent = 'Copied!';
        button.classList.add('copied');
        setTimeout(() => {
            button.textContent = 'Copy';
            button.classList.remove('copied');
        }, 2000);
    } catch (error) {
        console.error('Failed to copy:', error);
        // Fallback for older browsers
        const textarea = document.createElement('textarea');
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        button.textContent = 'Copied!';
        button.classList.add('copied');
        setTimeout(() => {
            button.textContent = 'Copy';
            button.classList.remove('copied');
        }, 2000);
    }
}

// Utility functions
function showLoading() {
    moduleList.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            <p>Loading modules...</p>
        </div>
    `;
}

function showError(message) {
    moduleList.innerHTML = `
        <div class="empty-state">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
            <h3>Error</h3>
            <p>${escapeHtml(message)}</p>
        </div>
    `;
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    }
    if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function formatDate(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now - date;
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    
    if (diffDays === 0) return 'today';
    if (diffDays === 1) return 'yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
    if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
    return `${Math.floor(diffDays / 365)} years ago`;
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}
