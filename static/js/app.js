// State
let users = [];
let selectedUserId = null;
let selectedUserEmail = null;
let accessList = [];
let searchResults = [];

// DOM Elements
const usersList = document.getElementById('users-list');
const accessListEl = document.getElementById('access-list');
const selectedUserName = document.getElementById('selected-user-name');
const addGroupsBtn = document.getElementById('add-groups-btn');
const userFilter = document.getElementById('user-filter');
const userSort = document.getElementById('user-sort');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadUsers();

    // Filter users on input
    userFilter.addEventListener('input', debounce(() => loadUsers(), 300));

    // Sort users on change
    userSort.addEventListener('change', () => loadUsers());

    // Search on enter
    document.getElementById('group-search-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            searchGroups();
        }
    });
});

// Debounce helper
function debounce(fn, delay) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn.apply(this, args), delay);
    };
}

// API calls
async function api(url, options = {}) {
    const response = await fetch(url, {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        ...options
    });

    if (!response.ok) {
        const text = await response.text();
        throw new Error(text || 'Request failed');
    }

    if (response.status === 204) {
        return null;
    }

    return response.json();
}

// Load users
async function loadUsers() {
    const filter = userFilter.value;
    const [sortField, sortDir] = userSort.value.split('-');

    try {
        let url = `/api/users?sort=${sortField}&dir=${sortDir}`;
        if (filter) {
            url += `&filter=${encodeURIComponent(filter)}`;
        }

        users = await api(url);
        renderUsers();
    } catch (error) {
        console.error('Failed to load users:', error);
        usersList.innerHTML = '<div class="empty-state">Fout bij laden van gebruikers</div>';
    }
}

// Render users
function renderUsers() {
    if (!users || users.length === 0) {
        usersList.innerHTML = '<div class="empty-state">Geen gebruikers gevonden</div>';
        return;
    }

    usersList.innerHTML = users.map(user => `
        <div class="user-item ${user.id === selectedUserId ? 'selected' : ''}"
             onclick="selectUser(${user.id}, '${escapeHtml(user.email)}')">
            <span class="user-item-email">${escapeHtml(user.email)}</span>
            <div class="user-item-actions">
                <button class="btn btn-sm btn-secondary" onclick="event.stopPropagation(); showEditUserModal(${user.id}, '${escapeHtml(user.email)}')">Bewerken</button>
                <button class="btn btn-sm btn-danger" onclick="event.stopPropagation(); showDeleteUserModal(${user.id}, '${escapeHtml(user.email)}')">Verwijderen</button>
            </div>
        </div>
    `).join('');
}

// Select user
async function selectUser(userId, email) {
    selectedUserId = userId;
    selectedUserEmail = email;
    selectedUserName.textContent = email;
    addGroupsBtn.disabled = false;

    // Highlight selected user
    document.querySelectorAll('.user-item').forEach(el => el.classList.remove('selected'));
    event.currentTarget.classList.add('selected');

    // Load access
    await loadUserAccess(userId);
}

// Load user access
async function loadUserAccess(userId) {
    try {
        accessList = await api(`/api/users/${userId}/access`);
        renderAccessList();
    } catch (error) {
        console.error('Failed to load user access:', error);
        accessListEl.innerHTML = '<div class="empty-state">Fout bij laden van groepen</div>';
    }
}

// Render access list
function renderAccessList() {
    if (!accessList || accessList.length === 0) {
        accessListEl.innerHTML = '<div class="empty-state">Geen groepen toegewezen</div>';
        return;
    }

    accessListEl.innerHTML = accessList.map(access => `
        <div class="access-item">
            <div class="access-item-info">
                <div class="access-item-name">${escapeHtml(access.groupName)}</div>
                <div class="access-item-date">Toegevoegd: ${formatDate(access.creationDate)}</div>
            </div>
            <button class="btn btn-sm btn-danger" onclick="removeAccess(${access.id})">Verwijderen</button>
        </div>
    `).join('');
}

// User Modal functions
function showAddUserModal() {
    document.getElementById('user-modal-title').textContent = 'Gebruiker toevoegen';
    document.getElementById('user-modal-id').value = '';
    document.getElementById('user-modal-email').value = '';
    document.getElementById('user-modal').classList.add('active');
    document.getElementById('user-modal-email').focus();
}

function showEditUserModal(userId, email) {
    document.getElementById('user-modal-title').textContent = 'Gebruiker bewerken';
    document.getElementById('user-modal-id').value = userId;
    document.getElementById('user-modal-email').value = email;
    document.getElementById('user-modal').classList.add('active');
    document.getElementById('user-modal-email').focus();
}

function hideUserModal() {
    document.getElementById('user-modal').classList.remove('active');
}

async function saveUser() {
    const id = document.getElementById('user-modal-id').value;
    const email = document.getElementById('user-modal-email').value.trim();

    if (!email) {
        alert('Vul een e-mailadres in');
        return;
    }

    try {
        if (id) {
            // Update
            await api(`/api/users/${id}`, {
                method: 'PUT',
                body: JSON.stringify({ email })
            });
        } else {
            // Create
            await api('/api/users', {
                method: 'POST',
                body: JSON.stringify({ email })
            });
        }

        hideUserModal();
        await loadUsers();
    } catch (error) {
        alert('Fout: ' + error.message);
    }
}

// Delete User Modal functions
function showDeleteUserModal(userId, email) {
    document.getElementById('delete-user-id').value = userId;
    document.getElementById('delete-user-name').textContent = email;
    document.getElementById('delete-modal').classList.add('active');
}

function hideDeleteModal() {
    document.getElementById('delete-modal').classList.remove('active');
}

async function confirmDeleteUser() {
    const userId = document.getElementById('delete-user-id').value;

    try {
        await api(`/api/users/${userId}`, { method: 'DELETE' });
        hideDeleteModal();

        // Clear selection if deleted user was selected
        if (parseInt(userId) === selectedUserId) {
            selectedUserId = null;
            selectedUserEmail = null;
            selectedUserName.textContent = '-';
            addGroupsBtn.disabled = true;
            accessListEl.innerHTML = '';
        }

        await loadUsers();
    } catch (error) {
        alert('Fout: ' + error.message);
    }
}

// Search Modal functions
function showSearchModal() {
    document.getElementById('group-search-input').value = '';
    document.getElementById('search-results').innerHTML = '<div class="search-empty">Voer een zoekterm in</div>';
    searchResults = [];
    document.getElementById('search-modal').classList.add('active');
    document.getElementById('group-search-input').focus();
}

function hideSearchModal() {
    document.getElementById('search-modal').classList.remove('active');
}

async function searchGroups() {
    const query = document.getElementById('group-search-input').value.trim();
    const resultsEl = document.getElementById('search-results');

    if (!query) {
        resultsEl.innerHTML = '<div class="search-empty">Voer een zoekterm in</div>';
        return;
    }

    resultsEl.innerHTML = '<div class="loading">Zoeken...</div>';

    try {
        searchResults = await api(`/api/groups/search?q=${encodeURIComponent(query)}`);

        if (!searchResults || searchResults.length === 0) {
            resultsEl.innerHTML = '<div class="search-empty">Geen groepen gevonden</div>';
            return;
        }

        // Filter out groups the user already has
        const existingBkeys = new Set((accessList || []).map(a => a.groupBkey));
        const filteredResults = searchResults.filter(r => !existingBkeys.has(r.groupBkey));

        if (filteredResults.length === 0) {
            resultsEl.innerHTML = '<div class="search-empty">Alle gevonden groepen zijn al toegewezen</div>';
            return;
        }

        resultsEl.innerHTML = filteredResults.map(result => `
            <label class="search-result-item">
                <input type="checkbox" value="${result.groupBkey}">
                <div class="search-result-info">
                    <div class="search-result-name">${escapeHtml(result.groupName)}</div>
                    <div class="search-result-match">Gevonden in: ${result.matchedOn}</div>
                </div>
            </label>
        `).join('');
    } catch (error) {
        resultsEl.innerHTML = '<div class="search-empty">Fout bij zoeken: ' + escapeHtml(error.message) + '</div>';
    }
}

async function addSelectedGroups() {
    const checkboxes = document.querySelectorAll('#search-results input[type="checkbox"]:checked');
    const groupBkeys = Array.from(checkboxes).map(cb => parseInt(cb.value));

    if (groupBkeys.length === 0) {
        alert('Selecteer minimaal één groep');
        return;
    }

    try {
        await api(`/api/users/${selectedUserId}/access`, {
            method: 'POST',
            body: JSON.stringify({ groupBkeys })
        });

        hideSearchModal();
        await loadUserAccess(selectedUserId);
    } catch (error) {
        alert('Fout: ' + error.message);
    }
}

// Remove access (no confirmation needed per requirements)
async function removeAccess(accessId) {
    try {
        await api(`/api/access/${accessId}`, { method: 'DELETE' });
        await loadUserAccess(selectedUserId);
    } catch (error) {
        alert('Fout: ' + error.message);
    }
}

// Helpers
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('nl-NL', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    });
}
