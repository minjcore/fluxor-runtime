// Dashboard JavaScript Client
(function() {
    'use strict';

    const API_ENDPOINT = '/api/dashboard/metrics';
    const REFRESH_INTERVAL = 2000; // 2 seconds
    const HISTORY_LENGTH = 30; // 1 minute at 2 second intervals (reduced from 150)

    let throughputChart = null;
    let metricsHistory = [];
    let lastMetrics = null; // Track previous metrics for RPS and network calculation
    let currentMetricType = 'rps'; // Current metric type: 'tasks', 'rps', 'network' (default: 'rps')

    // Initialize chart
    function initChart() {
        const ctx = document.getElementById('throughput-chart');
        if (!ctx) return;

        throughputChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'RPS (requests/sec)',
                    id: 'throughput-dataset',
                    data: [],
                    borderColor: 'rgb(59, 130, 246)',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: true,
                        labels: {
                            color: '#e2e8f0'
                        }
                    }
                },
                scales: {
                    x: {
                        ticks: {
                            color: '#94a3b8',
                            maxRotation: 45,
                            minRotation: 45,
                            maxTicksLimit: 10
                        },
                        grid: {
                            color: 'rgba(148, 163, 184, 0.1)'
                        }
                    },
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Value'
                        },
                        ticks: {
                            color: '#94a3b8',
                            callback: function(value) {
                                if (currentMetricType === 'network') {
                                    return formatBytes(value) + '/s';
                                }
                                return value;
                            }
                        },
                        grid: {
                            color: 'rgba(148, 163, 184, 0.1)'
                        }
                    }
                }
            }
        });
    }

    // Fetch metrics from API
    async function fetchMetrics() {
        try {
            const response = await fetch(API_ENDPOINT);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const data = await response.json();
            updateDashboard(data);
            updateRefreshIndicator(true);
        } catch (error) {
            console.error('Error fetching metrics:', error);
            updateRefreshIndicator(false);
            showError('Failed to fetch metrics: ' + error.message);
        }
    }

    // Update dashboard UI with new metrics
    function updateDashboard(metrics) {
        // Calculate aggregated metrics
        let totalQueued = 0;
        let totalThroughput = 0;
        let totalWorkerUtil = 0;
        let workerUtilCount = 0;

        // Aggregate executor metrics
        metrics.executors.forEach(exec => {
            totalQueued += exec.queuedTasks;
            totalThroughput += exec.throughput;
            if (exec.workerUtilization > 0) {
                totalWorkerUtil += exec.workerUtilization;
                workerUtilCount++;
            }
        });

        // Aggregate worker pool metrics
        metrics.workerPools.forEach(pool => {
            totalQueued += pool.queuedTasks;
            totalThroughput += pool.throughput;
            if (pool.workerUtilization > 0) {
                totalWorkerUtil += pool.workerUtilization;
                workerUtilCount++;
            }
        });

        const avgWorkerUtil = workerUtilCount > 0 ? totalWorkerUtil / workerUtilCount : 0;

        // Update metric cards
        updateMetricCard('queue-value', totalQueued.toLocaleString());
        updateMetricCard('queue-detail', `${metrics.executors.length + metrics.workerPools.length} instances`);

        updateMetricCard('throughput-value', totalThroughput.toFixed(1));
        updateMetricCard('throughput-detail', 'tasks/sec');

        updateMetricCard('worker-value', avgWorkerUtil.toFixed(1) + '%');
        updateMetricCard('worker-detail', `${workerUtilCount} active`);

        // Update allocation rate if available
        if (metrics.runtime && metrics.runtime.allocRate) {
            updateMetricCard('alloc-rate-value', metrics.runtime.allocRate.toFixed(0));
            updateMetricCard('alloc-rate-detail', 'allocs/sec');
        }

        // Calculate metric value based on selected type
        let chartValue = totalThroughput; // Default: tasks/sec
        
        if (currentMetricType === 'rps') {
            // Calculate RPS from HTTP servers
            chartValue = calculateRPS(metrics.httpServers, metrics.timestamp);
        } else if (currentMetricType === 'network') {
            // Calculate network throughput
            chartValue = calculateNetworkThroughput(metrics.httpServers, metrics.timestamp);
        }
        
        // Update chart
        updateChart(chartValue, metrics.timestamp);

        // Update statistics table
        updateStatsTable(metrics);

        // Update HTTP server metrics
        if (metrics.httpServers && metrics.httpServers.length > 0) {
            updateHTTPMetrics(metrics.httpServers, metrics.timestamp);
            updateHTTPServerTable(metrics.httpServers);
        }
        
        // Store current metrics for next RPS and network calculation
        lastMetrics = {
            timestamp: metrics.timestamp,
            httpServers: metrics.httpServers ? JSON.parse(JSON.stringify(metrics.httpServers)) : null
        };

        // Update profiling metrics
        if (metrics.profiling) {
            updateProfilingMetrics(metrics.profiling);
        }

        // Update runtime metrics
        if (metrics.runtime) {
            updateRuntimeMetrics(metrics.runtime);
        }

        // Update last update time
        const timestamp = new Date(metrics.timestamp);
        document.getElementById('last-update').textContent = 
            'Last update: ' + timestamp.toLocaleTimeString();
    }

    // Update metric card value
    function updateMetricCard(elementId, value) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = value;
        }
    }

    // Calculate RPS from HTTP servers
    function calculateRPS(httpServers, currentTimestamp) {
        if (!httpServers || httpServers.length === 0) {
            return 0;
        }
        
        if (!lastMetrics || !lastMetrics.timestamp || !lastMetrics.httpServers) {
            return 0;
        }
        
        const deltaTime = (new Date(currentTimestamp) - new Date(lastMetrics.timestamp)) / 1000; // seconds
        if (deltaTime <= 0) {
            return 0;
        }
        
        let totalRPS = 0;
        httpServers.forEach((server, index) => {
            const lastServer = lastMetrics.httpServers[index];
            if (lastServer) {
                const deltaRequests = (server.totalRequests || 0) - (lastServer.totalRequests || 0);
                const rps = deltaRequests / deltaTime;
                if (rps > 0 && rps < 1000000) { // Sanity check
                    totalRPS += rps;
                }
            }
        });
        
        return Math.round(totalRPS);
    }

    // Calculate network throughput (bytes/sec)
    function calculateNetworkThroughput(httpServers, currentTimestamp) {
        if (!httpServers || httpServers.length === 0) {
            return 0;
        }
        
        if (!lastMetrics || !lastMetrics.timestamp || !lastMetrics.httpServers) {
            return 0;
        }
        
        const deltaTime = (new Date(currentTimestamp) - new Date(lastMetrics.timestamp)) / 1000; // seconds
        if (deltaTime <= 0) {
            return 0;
        }
        
        let totalBytes = 0;
        httpServers.forEach((server, index) => {
            const lastServer = lastMetrics.httpServers[index];
            if (lastServer) {
                const deltaBytesSent = (server.bytesSent || 0) - (lastServer.bytesSent || 0);
                const deltaBytesReceived = (server.bytesReceived || 0) - (lastServer.bytesReceived || 0);
                const totalDeltaBytes = deltaBytesSent + deltaBytesReceived;
                if (totalDeltaBytes > 0 && totalDeltaBytes < 1e12) { // Sanity check (1TB)
                    totalBytes += totalDeltaBytes;
                }
            }
        });
        
        return totalBytes / deltaTime; // bytes/sec
    }

    // Update throughput chart
    function updateChart(value, timestamp) {
        if (!throughputChart) {
            console.error('Chart not initialized');
            return;
        }

        const timeStr = new Date(timestamp).toLocaleTimeString();
        
        // Add to history
        metricsHistory.push({
            time: timeStr,
            value: value
        });

        // Limit history length
        if (metricsHistory.length > HISTORY_LENGTH) {
            metricsHistory.shift();
        }

        // Update chart
        throughputChart.data.labels = metricsHistory.map(m => m.time);
        throughputChart.data.datasets[0].data = metricsHistory.map(m => m.value);
        
        // Update label if needed
        updateChartLabel();
        
        // Force update with animation
        throughputChart.update();
    }

    // Update statistics table
    function updateStatsTable(metrics) {
        const tbody = document.getElementById('stats-body');
        if (!tbody) return;

        // Clear existing rows
        tbody.innerHTML = '';

        // Add executor rows
        metrics.executors.forEach(exec => {
            const row = createStatsRow(exec.id, 'Executor', exec);
            tbody.appendChild(row);
        });

        // Add worker pool rows
        metrics.workerPools.forEach(pool => {
            const row = createStatsRow(pool.id, 'WorkerPool', pool);
            tbody.appendChild(row);
        });

        if (tbody.children.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="8" class="no-data">No executors or worker pools registered</td>';
            tbody.appendChild(row);
        }
    }

    // Create statistics table row
    function createStatsRow(id, type, metrics) {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${escapeHtml(id)}</td>
            <td>${escapeHtml(type)}</td>
            <td>${metrics.queuedTasks.toLocaleString()}</td>
            <td>${metrics.queueUtilization.toFixed(1)}%</td>
            <td>${metrics.throughput.toFixed(1)}</td>
            <td>${metrics.workerUtilization.toFixed(1)}%</td>
            <td>${metrics.completedTasks.toLocaleString()}</td>
            <td>${metrics.rejectedTasks.toLocaleString()}</td>
        `;
        return row;
    }

    // Escape HTML to prevent XSS
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Update refresh indicator
    function updateRefreshIndicator(success) {
        const indicator = document.getElementById('refresh-indicator');
        if (indicator) {
            indicator.className = 'refresh-indicator ' + (success ? 'success' : 'error');
        }
    }

    // Calculate RPS for a single server
    function calculateRPSForServer(server) {
        if (!lastMetrics || !lastMetrics.httpServers) {
            return 0;
        }
        
        const lastServer = lastMetrics.httpServers.find(s => s.name === server.name);
        if (!lastServer || !lastMetrics.timestamp) {
            return 0;
        }
        
        const currentTime = new Date();
        const lastTime = new Date(lastMetrics.timestamp);
        const deltaTime = (currentTime - lastTime) / 1000; // seconds
        
        if (deltaTime <= 0) {
            return 0;
        }
        
        const deltaRequests = (server.totalRequests || 0) - (lastServer.totalRequests || 0);
        const rps = deltaRequests / deltaTime;
        
        // Sanity check
        if (rps < 0 || rps > 1000000) {
            return 0;
        }
        
        return rps;
    }

    // Update HTTP server metrics
    function updateHTTPMetrics(httpServers, currentTimestamp) {
        let totalRPS = 0;
        let totalQueued = 0;
        let totalRejected = 0;

        if (lastMetrics && lastMetrics.timestamp) {
            // Calculate RPS based on delta requests / delta time
            const deltaTime = (new Date(currentTimestamp) - new Date(lastMetrics.timestamp)) / 1000; // seconds
            
            if (deltaTime > 0) {
                httpServers.forEach((server, index) => {
                    const lastServer = lastMetrics.httpServers && lastMetrics.httpServers[index];
                    if (lastServer) {
                        const deltaRequests = (server.totalRequests || 0) - (lastServer.totalRequests || 0);
                        const rps = deltaRequests / deltaTime;
                        if (rps > 0 && rps < 1000000) { // Sanity check: RPS should be reasonable
                            totalRPS += rps;
                        }
                    }
                });
            }
        }

        // If no previous metrics or calculation failed, show 0
        totalRPS = Math.round(totalRPS);

        httpServers.forEach(server => {
            totalQueued += server.queuedRequests || 0;
            totalRejected += server.rejectedRequests || 0;
        });

        updateMetricCard('rps-value', totalRPS.toLocaleString());
        updateMetricCard('rps-detail', `${httpServers.length} server(s)`);
    }

    // Update HTTP server statistics table
    function updateHTTPServerTable(httpServers) {
        const tbody = document.getElementById('http-stats-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        httpServers.forEach(server => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${escapeHtml(server.name)}</td>
                <td>${calculateRPSForServer(server).toFixed(0)}</td>
                <td>${(server.queuedRequests || 0).toLocaleString()}</td>
                <td>${(server.queueUtilization || 0).toFixed(1)}%</td>
                <td>${(server.rejectedRequests || 0).toLocaleString()}</td>
                <td>${(server.currentCCU || 0).toLocaleString()}</td>
                <td>${(server.ccuUtilization || 0).toFixed(1)}%</td>
                <td>${(server.workers || 0).toLocaleString()}</td>
            `;
            tbody.appendChild(row);
        });

        if (tbody.children.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="8" class="no-data">No HTTP servers registered</td>';
            tbody.appendChild(row);
        }
    }

    // Update profiling metrics
    function updateProfilingMetrics(profiling) {
        const section = document.getElementById('profiling-section');
        if (section) {
            section.style.display = 'block';
        }

        // Update work classification
        const workClassDiv = document.getElementById('work-classification');
        if (workClassDiv && profiling.workClassification) {
            const wc = profiling.workClassification;
            workClassDiv.innerHTML = `
                <div><strong>IO-Bound:</strong> ${wc.ioBound.activeWorkers} workers, ${wc.ioBound.utilization.toFixed(1)}% util</div>
                <div><strong>CPU-Bound:</strong> ${wc.cpuBound.activeWorkers} workers, ${wc.cpuBound.utilization.toFixed(1)}% util</div>
                <div><strong>Mixed:</strong> ${wc.mixed.activeWorkers} workers ${wc.mixed.warning ? '<span style="color: #ef4444;">⚠️ ' + escapeHtml(wc.mixed.warning) + '</span>' : ''}</div>
            `;
        }

        // Update bottlenecks
        const bottlenecksDiv = document.getElementById('bottlenecks');
        if (bottlenecksDiv && profiling.bottlenecks) {
            if (profiling.bottlenecks.length === 0) {
                bottlenecksDiv.innerHTML = '<div style="color: #10b981;">✓ No bottlenecks detected</div>';
            } else {
                let html = '<ul style="list-style: none; padding: 0;">';
                profiling.bottlenecks.forEach(b => {
                    const severityColor = b.severity === 'critical' ? '#ef4444' : 
                                         b.severity === 'high' ? '#f59e0b' :
                                         b.severity === 'medium' ? '#fbbf24' : '#94a3b8';
                    html += `<li style="margin: 8px 0; padding: 8px; background: #0f172a; border-left: 3px solid ${severityColor};">
                        <strong style="color: ${severityColor};">[${b.severity.toUpperCase()}]</strong> ${escapeHtml(b.type)}<br>
                        <small style="color: #94a3b8;">${escapeHtml(b.description)}</small><br>
                        <small style="color: #64748b;">💡 ${escapeHtml(b.recommendation)}</small>
                    </li>`;
                });
                html += '</ul>';
                bottlenecksDiv.innerHTML = html;
            }
        }

        // Update goroutines
        if (profiling.goroutines) {
            updateMetricCard('goroutines-value', profiling.goroutines.total.toLocaleString());
            const byState = profiling.goroutines.byState || {};
            const states = [];
            if (byState.running) states.push(`${byState.running} running`);
            if (byState.waiting) states.push(`${byState.waiting} waiting`);
            if (byState.blocked) states.push(`${byState.blocked} blocked`);
            updateMetricCard('goroutines-detail', states.join(', ') || 'N/A');
        }
    }

    // Update runtime metrics
    function updateRuntimeMetrics(runtime) {
        const tbody = document.getElementById('runtime-stats-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        const metrics = [
            { name: 'Goroutines', value: runtime.goroutines.toLocaleString(), desc: 'Current number of goroutines' },
            { name: 'CPU Cores', value: runtime.numCPU.toString(), desc: 'Number of CPU cores' },
            { name: 'GOMAXPROCS', value: runtime.goMaxProcs.toString(), desc: 'Maximum number of OS threads' },
            { name: 'OS Threads', value: (runtime.osThreads || runtime.goMaxProcs).toString(), desc: 'Actual OS threads (may be > GOMAXPROCS due to blocking I/O, CGO, runtime threads)' },
            { name: 'Memory Allocated', value: formatBytes(runtime.alloc), desc: 'Currently allocated memory' },
            { name: 'Total Allocated', value: formatBytes(runtime.totalAlloc), desc: 'Total memory allocated since start' },
            { name: 'System Memory', value: formatBytes(runtime.sys), desc: 'Memory obtained from OS' },
            { name: 'Total Mallocs', value: runtime.mallocs.toLocaleString(), desc: 'Total number of allocations' },
            { name: 'Total Frees', value: runtime.frees.toLocaleString(), desc: 'Total number of frees' },
            { name: 'GC Cycles', value: runtime.numGC.toString(), desc: 'Number of GC cycles' },
            { name: 'Alloc Rate', value: (runtime.allocRate || 0).toFixed(0) + '/sec', desc: 'Allocations per second' },
            { name: 'GC Rate', value: (runtime.gcRate || 0).toFixed(2) + '/sec', desc: 'GC cycles per second' },
            { name: 'Last GC', value: runtime.lastGC ? new Date(runtime.lastGC).toLocaleTimeString() : 'N/A', desc: 'Last garbage collection time' },
        ];

        metrics.forEach(m => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><strong>${escapeHtml(m.name)}</strong></td>
                <td>${escapeHtml(m.value)}</td>
                <td style="color: #94a3b8; font-size: 0.875rem;">${escapeHtml(m.desc)}</td>
            `;
            tbody.appendChild(row);
        });
    }

    // Format bytes to human-readable format
    function formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    }

    // Show error message
    function showError(message) {
        console.error(message);
        // Could add UI error display here
    }

    // Start auto-refresh
    function startAutoRefresh() {
        // Initial fetch
        fetchMetrics();
        
        // Set up interval
        setInterval(fetchMetrics, REFRESH_INTERVAL);
    }

    // Setup metric type selector
    function setupMetricSelector() {
        const selector = document.getElementById('metric-type-selector');
        if (selector) {
            selector.addEventListener('change', function(e) {
                currentMetricType = e.target.value;
                // Clear history when switching metric type
                metricsHistory = [];
                // Update chart label
                if (throughputChart) {
                    updateChartLabel();
                }
            });
        }
    }

    // Update chart label based on current metric type
    function updateChartLabel() {
        if (!throughputChart) return;
        
        const labels = {
            'tasks': 'Throughput (tasks/sec)',
            'rps': 'RPS (requests/sec)',
            'network': 'Network (bytes/sec)'
        };
        
        throughputChart.data.datasets[0].label = labels[currentMetricType] || labels['rps'];
        throughputChart.update('none'); // Update without animation
    }

    // Initialize dashboard when DOM is ready
    function init() {
        initChart();
        setupMetricSelector();
        startAutoRefresh();
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();

