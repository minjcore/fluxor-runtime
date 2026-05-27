package views

// GetAccountantDashboardHTML returns the HTML for the accountant dashboard
func GetAccountantDashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Accountant Dashboard - Ledger Management</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        .header {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            margin-bottom: 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .header h1 {
            color: #333;
            margin-bottom: 10px;
        }
        .header .nav {
            display: flex;
            gap: 10px;
        }
        .btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            text-decoration: none;
            display: inline-block;
        }
        .btn:hover {
            background: #5568d3;
        }
        .btn-secondary {
            background: #6c757d;
        }
        .btn-secondary:hover {
            background: #5a6268;
        }
        .card {
            background: white;
            padding: 25px;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            margin-bottom: 20px;
        }
        .card h2 {
            color: #333;
            margin-bottom: 15px;
            font-size: 1.5em;
        }
        .filters {
            display: flex;
            gap: 15px;
            flex-wrap: wrap;
            margin-bottom: 20px;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 8px;
        }
        .filter-group {
            display: flex;
            flex-direction: column;
            gap: 5px;
        }
        .filter-group label {
            font-size: 0.9em;
            color: #666;
            font-weight: 500;
        }
        .filter-group input,
        .filter-group select {
            padding: 8px 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 14px;
        }
        .filter-group input:focus,
        .filter-group select:focus {
            outline: none;
            border-color: #667eea;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }
        .stat-box {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }
        .stat-label {
            font-size: 0.9em;
            opacity: 0.9;
            margin-bottom: 8px;
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
        }
        .stat-box.success {
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
        }
        .stat-box.warning {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        }
        .stat-box.info {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 15px;
        }
        table th,
        table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #e0e0e0;
        }
        table th {
            background: #f8f9fa;
            font-weight: 600;
            color: #333;
            position: sticky;
            top: 0;
        }
        table tr:hover {
            background: #f8f9fa;
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 500;
        }
        .badge.posted {
            background: #d4edda;
            color: #155724;
        }
        .badge.pending {
            background: #fff3cd;
            color: #856404;
        }
        .badge.reversed {
            background: #f8d7da;
            color: #721c24;
        }
        .badge.asset {
            background: #cfe2ff;
            color: #084298;
        }
        .badge.liability {
            background: #fff3cd;
            color: #856404;
        }
        .badge.revenue {
            background: #d1e7dd;
            color: #0f5132;
        }
        .badge.expense {
            background: #f8d7da;
            color: #721c24;
        }
        .badge.equity {
            background: #d0bdf4;
            color: #5a189a;
        }
        .chart-container {
            position: relative;
            height: 300px;
            margin-top: 20px;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
        .tabs {
            display: flex;
            gap: 10px;
            border-bottom: 2px solid #e0e0e0;
            margin-bottom: 20px;
        }
        .tab {
            padding: 12px 24px;
            background: none;
            border: none;
            cursor: pointer;
            font-size: 16px;
            color: #666;
            border-bottom: 3px solid transparent;
            transition: all 0.3s;
        }
        .tab.active {
            color: #667eea;
            border-bottom-color: #667eea;
            font-weight: 600;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: block;
        }
        .text-right {
            text-align: right;
        }
        .text-green {
            color: #28a745;
        }
        .text-red {
            color: #dc3545;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div>
                <h1>📊 Accountant Dashboard</h1>
                <p>Double-Entry Bookkeeping & Financial Reports</p>
            </div>
            <div class="nav">
                <a href="/dashboard" class="btn btn-secondary">← Back to Dashboard</a>
                <button class="btn" onclick="logout()">Logout</button>
            </div>
        </div>

        <div id="error-container"></div>

        <div class="card">
            <div class="stats-grid">
                <div class="stat-box" id="stat-total-accounts">
                    <div class="stat-label">Total Accounts</div>
                    <div class="stat-value">-</div>
                </div>
                <div class="stat-box info" id="stat-total-debit">
                    <div class="stat-label">Total Debit</div>
                    <div class="stat-value">$0.00</div>
                </div>
                <div class="stat-box success" id="stat-total-credit">
                    <div class="stat-label">Total Credit</div>
                    <div class="stat-value">$0.00</div>
                </div>
                <div class="stat-box" id="stat-balanced">
                    <div class="stat-label">Balanced</div>
                    <div class="stat-value">-</div>
                </div>
            </div>
        </div>

        <div class="card">
            <div class="filters">
                <div class="filter-group">
                    <label>Start Date</label>
                    <input type="date" id="filter-start-date">
                </div>
                <div class="filter-group">
                    <label>End Date</label>
                    <input type="date" id="filter-end-date">
                </div>
                <div class="filter-group">
                    <label>Journal Status</label>
                    <select id="filter-status">
                        <option value="">All</option>
                        <option value="posted">Posted</option>
                        <option value="pending">Pending</option>
                        <option value="reversed">Reversed</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>Reference Type</label>
                    <select id="filter-reference-type">
                        <option value="">All</option>
                        <option value="wallet_topup">Wallet Top-up</option>
                        <option value="purchase">Purchase</option>
                        <option value="freeze">Freeze</option>
                        <option value="unfreeze">Unfreeze</option>
                        <option value="refund">Refund</option>
                        <option value="adjustment">Adjustment</option>
                    </select>
                </div>
                <div class="filter-group" style="justify-content: flex-end;">
                    <label>&nbsp;</label>
                    <button class="btn" onclick="applyFilters()">Apply Filters</button>
                </div>
            </div>

            <div class="tabs">
                <button class="tab active" onclick="switchTab('trial-balance')">Trial Balance</button>
                <button class="tab" onclick="switchTab('account-balances')">Account Balances</button>
                <button class="tab" onclick="switchTab('journals')">Journals</button>
                <button class="tab" onclick="switchTab('chart')">Charts</button>
            </div>

            <div id="tab-trial-balance" class="tab-content active">
                <h2>Trial Balance</h2>
                <div id="trial-balance-content" class="loading">Loading...</div>
            </div>

            <div id="tab-account-balances" class="tab-content">
                <h2>Account Balances</h2>
                <div id="account-balances-content" class="loading">Loading...</div>
            </div>

            <div id="tab-journals" class="tab-content">
                <h2>Journal Entries</h2>
                <div id="journals-content" class="loading">Loading...</div>
            </div>

            <div id="tab-chart" class="tab-content">
                <h2>Financial Overview</h2>
                <div class="chart-container">
                    <canvas id="balance-chart"></canvas>
                </div>
            </div>
        </div>
    </div>

    <script>
        const API_BASE = '/api/accountant';
        let balanceChart = null;

        // Get token from localStorage
        function getToken() {
            return localStorage.getItem('auth_token');
        }

        // Set token in Authorization header
        function authHeaders() {
            return {
                'Authorization': 'Bearer ' + getToken(),
                'Content-Type': 'application/json'
            };
        }

        // Logout function
        function logout() {
            localStorage.removeItem('auth_token');
            window.location.href = '/login';
        }

        // Show error message
        function showError(message) {
            const container = document.getElementById('error-container');
            container.innerHTML = '<div class="error">' + message + '</div>';
            setTimeout(() => {
                container.innerHTML = '';
            }, 5000);
        }

        // Format currency
        function formatCurrency(value) {
            return '$' + parseFloat(value).toFixed(2);
        }

        // Format date
        function formatDate(dateStr) {
            if (!dateStr) return '-';
            const date = new Date(dateStr);
            return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
        }

        // Switch tabs
        function switchTab(tabName) {
            // Hide all tabs
            document.querySelectorAll('.tab-content').forEach(tab => {
                tab.classList.remove('active');
            });
            document.querySelectorAll('.tab').forEach(tab => {
                tab.classList.remove('active');
            });

            // Show selected tab
            document.getElementById('tab-' + tabName).classList.add('active');
            event.target.classList.add('active');

            // Load data for selected tab
            if (tabName === 'trial-balance') {
                loadTrialBalance();
            } else if (tabName === 'account-balances') {
                loadAccountBalances();
            } else if (tabName === 'journals') {
                loadJournals();
            } else if (tabName === 'chart') {
                loadChart();
            }
        }

        // Apply filters
        function applyFilters() {
            switchTab(document.querySelector('.tab.active').textContent.toLowerCase().replace(' ', '-'));
        }

        // Get filter params
        function getFilterParams() {
            const params = new URLSearchParams();
            const startDate = document.getElementById('filter-start-date').value;
            const endDate = document.getElementById('filter-end-date').value;
            if (startDate) params.append('start_date', startDate);
            if (endDate) params.append('end_date', endDate);
            return params.toString();
        }

        // Load Trial Balance
        async function loadTrialBalance() {
            const content = document.getElementById('trial-balance-content');
            content.innerHTML = '<div class="loading">Loading trial balance...</div>';

            try {
                const params = getFilterParams();
                const response = await fetch(API_BASE + '/trial-balance' + (params ? '?' + params : ''), {
                    headers: authHeaders()
                });

                if (!response.ok) {
                    if (response.status === 401) {
                        logout();
                        return;
                    }
                    throw new Error('Failed to load trial balance');
                }

                const data = await response.json();

                // Update stats
                document.getElementById('stat-total-debit').querySelector('.stat-value').textContent = formatCurrency(data.total_debit || 0);
                document.getElementById('stat-total-credit').querySelector('.stat-value').textContent = formatCurrency(data.total_credit || 0);
                const balancedEl = document.getElementById('stat-balanced').querySelector('.stat-value');
                balancedEl.textContent = data.is_balanced ? '✓ Yes' : '✗ No';
                balancedEl.parentElement.className = 'stat-box ' + (data.is_balanced ? 'success' : 'warning');

                // Render table
                let html = '<table><thead><tr><th>Account Code</th><th>Account Name</th><th>Type</th><th class="text-right">Debit</th><th class="text-right">Credit</th><th class="text-right">Balance</th></tr></thead><tbody>';
                
                if (data.accounts && data.accounts.length > 0) {
                    data.accounts.forEach(acc => {
                        const balanceClass = acc.balance >= 0 ? 'text-green' : 'text-red';
                        html += '<tr>';
                        html += '<td><strong>' + acc.account_code + '</strong></td>';
                        html += '<td>' + acc.account_name + '</td>';
                        html += '<td><span class="badge ' + acc.account_type + '">' + acc.account_type + '</span></td>';
                        html += '<td class="text-right">' + formatCurrency(acc.total_debit) + '</td>';
                        html += '<td class="text-right">' + formatCurrency(acc.total_credit) + '</td>';
                        html += '<td class="text-right ' + balanceClass + '"><strong>' + formatCurrency(acc.balance) + '</strong></td>';
                        html += '</tr>';
                    });
                } else {
                    html += '<tr><td colspan="6" style="text-align: center; padding: 40px;">No accounts found</td></tr>';
                }

                html += '</tbody><tfoot><tr style="background: #f8f9fa; font-weight: bold;"><td colspan="3">Total</td><td class="text-right">' + formatCurrency(data.total_debit || 0) + '</td><td class="text-right">' + formatCurrency(data.total_credit || 0) + '</td><td></td></tr></tfoot></table>';

                content.innerHTML = html;
            } catch (error) {
                content.innerHTML = '<div class="error">Error: ' + error.message + '</div>';
                showError(error.message);
            }
        }

        // Load Account Balances
        async function loadAccountBalances() {
            const content = document.getElementById('account-balances-content');
            content.innerHTML = '<div class="loading">Loading account balances...</div>';

            try {
                const params = getFilterParams();
                const response = await fetch(API_BASE + '/account-balances' + (params ? '?' + params : ''), {
                    headers: authHeaders()
                });

                if (!response.ok) {
                    if (response.status === 401) {
                        logout();
                        return;
                    }
                    throw new Error('Failed to load account balances');
                }

                const data = await response.json();

                // Update total accounts stat
                document.getElementById('stat-total-accounts').querySelector('.stat-value').textContent = data.length || 0;

                let html = '<table><thead><tr><th>Account Code</th><th>Account Name</th><th>Type</th><th class="text-right">Debit</th><th class="text-right">Credit</th><th class="text-right">Balance</th></tr></thead><tbody>';
                
                if (data && data.length > 0) {
                    data.forEach(acc => {
                        const balanceClass = acc.balance >= 0 ? 'text-green' : 'text-red';
                        html += '<tr>';
                        html += '<td><strong>' + acc.account_code + '</strong></td>';
                        html += '<td>' + acc.account_name + '</td>';
                        html += '<td><span class="badge ' + acc.account_type + '">' + acc.account_type + '</span></td>';
                        html += '<td class="text-right">' + formatCurrency(acc.total_debit) + '</td>';
                        html += '<td class="text-right">' + formatCurrency(acc.total_credit) + '</td>';
                        html += '<td class="text-right ' + balanceClass + '"><strong>' + formatCurrency(acc.balance) + '</strong></td>';
                        html += '</tr>';
                    });
                } else {
                    html += '<tr><td colspan="6" style="text-align: center; padding: 40px;">No accounts found</td></tr>';
                }

                html += '</tbody></table>';

                content.innerHTML = html;
            } catch (error) {
                content.innerHTML = '<div class="error">Error: ' + error.message + '</div>';
                showError(error.message);
            }
        }

        // Load Journals
        async function loadJournals() {
            const content = document.getElementById('journals-content');
            content.innerHTML = '<div class="loading">Loading journals...</div>';

            try {
                const params = new URLSearchParams();
                const startDate = document.getElementById('filter-start-date').value;
                const endDate = document.getElementById('filter-end-date').value;
                const status = document.getElementById('filter-status').value;
                const refType = document.getElementById('filter-reference-type').value;
                
                if (startDate) params.append('start_date', startDate);
                if (endDate) params.append('end_date', endDate);
                if (status) params.append('status', status);
                if (refType) params.append('reference_type', refType);
                params.append('limit', '100');

                const response = await fetch(API_BASE + '/journals' + (params.toString() ? '?' + params.toString() : ''), {
                    headers: authHeaders()
                });

                if (!response.ok) {
                    if (response.status === 401) {
                        logout();
                        return;
                    }
                    throw new Error('Failed to load journals');
                }

                const data = await response.json();

                let html = '<table><thead><tr><th>ID</th><th>Reference Type</th><th>Description</th><th>Status</th><th>Created At</th><th>Created By</th><th>Action</th></tr></thead><tbody>';
                
                if (data && data.length > 0) {
                    data.forEach(journal => {
                        const statusClass = journal.status || 'pending';
                        html += '<tr>';
                        html += '<td><strong>' + journal.id + '</strong></td>';
                        html += '<td>' + journal.reference_type + '</td>';
                        html += '<td>' + (journal.description || '-') + '</td>';
                        html += '<td><span class="badge ' + statusClass + '">' + (journal.status || 'pending') + '</span></td>';
                        html += '<td>' + formatDate(journal.created_at) + '</td>';
                        html += '<td>' + (journal.created_by || '-') + '</td>';
                        html += '<td><button class="btn" onclick="viewJournal(' + journal.id + ')">View</button></td>';
                        html += '</tr>';
                    });
                } else {
                    html += '<tr><td colspan="7" style="text-align: center; padding: 40px;">No journals found</td></tr>';
                }

                html += '</tbody></table>';

                content.innerHTML = html;
            } catch (error) {
                content.innerHTML = '<div class="error">Error: ' + error.message + '</div>';
                showError(error.message);
            }
        }

        // View journal details
        async function viewJournal(journalId) {
            try {
                const response = await fetch(API_BASE + '/journals/' + journalId, {
                    headers: authHeaders()
                });

                if (!response.ok) {
                    throw new Error('Failed to load journal');
                }

                const data = await response.json();
                const journal = data.journal;
                const entries = data.entries || [];

                let html = '<h3>Journal #' + journal.id + '</h3>';
                html += '<p><strong>Reference Type:</strong> ' + journal.reference_type + '</p>';
                html += '<p><strong>Description:</strong> ' + (journal.description || '-') + '</p>';
                html += '<p><strong>Status:</strong> <span class="badge ' + (journal.status || 'pending') + '">' + (journal.status || 'pending') + '</span></p>';
                html += '<p><strong>Created At:</strong> ' + formatDate(journal.created_at) + '</p>';
                
                html += '<h4 style="margin-top: 20px;">Ledger Entries</h4>';
                html += '<table><thead><tr><th>Account Code</th><th>Account Name</th><th class="text-right">Debit</th><th class="text-right">Credit</th></tr></thead><tbody>';
                
                entries.forEach(entry => {
                    html += '<tr>';
                    html += '<td>' + (entry.account ? entry.account.code : '-') + '</td>';
                    html += '<td>' + (entry.account ? entry.account.name : '-') + '</td>';
                    html += '<td class="text-right">' + (entry.debit > 0 ? formatCurrency(entry.debit) : '-') + '</td>';
                    html += '<td class="text-right">' + (entry.credit > 0 ? formatCurrency(entry.credit) : '-') + '</td>';
                    html += '</tr>';
                });

                html += '</tbody></table>';

                alert(html.replace(/<[^>]*>/g, '\\n')); // Simple alert, could use modal
            } catch (error) {
                showError('Error loading journal: ' + error.message);
            }
        }

        // Load Chart
        async function loadChart() {
            try {
                const response = await fetch(API_BASE + '/account-balances', {
                    headers: authHeaders()
                });

                if (!response.ok) {
                    throw new Error('Failed to load chart data');
                }

                const accounts = await response.json();
                
                // Group by account type
                const byType = {};
                accounts.forEach(acc => {
                    if (!byType[acc.account_type]) {
                        byType[acc.account_type] = 0;
                    }
                    byType[acc.account_type] += Math.abs(acc.balance);
                });

                const ctx = document.getElementById('balance-chart');
                if (balanceChart) {
                    balanceChart.destroy();
                }

                balanceChart = new Chart(ctx, {
                    type: 'doughnut',
                    data: {
                        labels: Object.keys(byType),
                        datasets: [{
                            data: Object.values(byType),
                            backgroundColor: [
                                '#667eea',
                                '#764ba2',
                                '#f093fb',
                                '#4facfe',
                                '#43e97b'
                            ]
                        }]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                position: 'bottom'
                            },
                            title: {
                                display: true,
                                text: 'Account Balances by Type'
                            }
                        }
                    }
                });
            } catch (error) {
                showError('Error loading chart: ' + error.message);
            }
        }

        // Initialize on page load
        document.addEventListener('DOMContentLoaded', function() {
            // Set default date range (last 30 days to today)
            const today = new Date();
            const thirtyDaysAgo = new Date();
            thirtyDaysAgo.setDate(today.getDate() - 30);
            
            document.getElementById('filter-start-date').value = thirtyDaysAgo.toISOString().split('T')[0];
            document.getElementById('filter-end-date').value = today.toISOString().split('T')[0];

            // Load initial data
            loadTrialBalance();
        });
    </script>
</body>
</html>`
}
