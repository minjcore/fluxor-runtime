package views

// GetDashboardHTML returns the HTML for the dashboard (protected)
func GetDashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PostgreSQL Demo - Web UI</title>
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
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            margin-bottom: 20px;
        }
        .header h1 {
            color: #333;
            margin-bottom: 10px;
        }
        .header p {
            color: #666;
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
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }
        .stat-box {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
        }
        .stat-label {
            font-size: 0.9em;
            color: #666;
            margin-bottom: 5px;
        }
        .stat-value {
            font-size: 1.8em;
            font-weight: bold;
            color: #333;
        }
        .query-section {
            margin-top: 20px;
        }
        .query-input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            margin-bottom: 10px;
            resize: vertical;
            min-height: 100px;
        }
        .query-input:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            transition: all 0.3s;
            margin-right: 10px;
        }
        .btn:hover {
            background: #5568d3;
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        .btn:active {
            transform: translateY(0);
        }
        .btn-secondary {
            background: #6c757d;
        }
        .btn-secondary:hover {
            background: #5a6268;
        }
        .results {
            margin-top: 20px;
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            max-height: 500px;
            overflow: auto;
        }
        .results table {
            width: 100%;
            border-collapse: collapse;
        }
        .results th,
        .results td {
            padding: 10px;
            text-align: left;
            border-bottom: 1px solid #dee2e6;
        }
        .results th {
            background: #667eea;
            color: white;
            position: sticky;
            top: 0;
        }
        .results tr:hover {
            background: #e9ecef;
        }
        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
        }
        .status-online {
            background: #28a745;
        }
        .status-offline {
            background: #dc3545;
        }
        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin-left: 10px;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 12px;
            border-radius: 8px;
            margin-top: 10px;
        }
        .success {
            background: #d4edda;
            color: #155724;
            padding: 12px;
            border-radius: 8px;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <div>
                    <h1>🐘 PostgreSQL Demo - Web UI</h1>
                    <p>Monitor and interact with your PostgreSQL database</p>
                </div>
                <div style="text-align: right;">
                    <span id="username-display" style="color: #666; margin-right: 15px;"></span>
                    <button class="btn btn-secondary" onclick="handleLogout()" style="margin: 0;">Logout</button>
                </div>
            </div>
        </div>

        <div class="card">
            <h2>💰 Wallet</h2>
            <div id="wallet-info" style="padding: 20px; background: #f8f9fa; border-radius: 8px; margin-bottom: 15px;">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                    <div style="flex: 1;">
                        <div style="font-size: 0.9em; color: #666; margin-bottom: 5px;">Available Balance</div>
                        <div id="wallet-balance" style="font-size: 2em; font-weight: bold; color: #667eea;">$0.00</div>
                        <div style="margin-top: 10px; font-size: 0.85em; color: #666;">
                            <div>Total: $<span id="wallet-total">0.00</span></div>
                            <div style="color: #ff9800;">Frozen: $<span id="wallet-frozen">0.00</span></div>
                        </div>
                    </div>
                    <div>
                        <button id="add-balance-btn" class="btn" style="padding: 10px 20px; font-size: 0.9em;">Add $100</button>
                    </div>
                </div>
                <div id="wallet-transactions" style="max-height: 200px; overflow-y: auto;">
                    <p style="color: #666; text-align: center;">Loading transactions...</p>
                </div>
            </div>
        </div>

        <div class="card">
            <h2>Database Status</h2>
            <div id="status-indicator">
                <span class="status-indicator status-offline"></span>
                <span>Checking...</span>
            </div>
            <div class="stats-grid" id="stats-grid">
                <div class="stat-box">
                    <div class="stat-label">Open Connections</div>
                    <div class="stat-value" id="open-connections">-</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">In Use</div>
                    <div class="stat-value" id="in-use">-</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">Idle</div>
                    <div class="stat-value" id="idle">-</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">Wait Count</div>
                    <div class="stat-value" id="wait-count">-</div>
                </div>
            </div>
        </div>

        <div class="card">
            <h2>🛒 Make a Purchase</h2>
            <div id="products-section">
                <div style="margin-bottom: 20px;">
                    <h3 style="margin-bottom: 15px;">Available Products</h3>
                    <div id="products-list" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 15px;"></div>
                </div>
                <div style="margin-top: 30px; padding-top: 20px; border-top: 2px solid #e0e0e0;">
                    <h3 style="margin-bottom: 15px;">Shopping Cart</h3>
                    <div id="cart-items"></div>
                    <div id="cart-total" style="margin-top: 15px; font-size: 1.2em; font-weight: bold; color: #333;"></div>
                    <button class="btn" onclick="makePurchase()" id="purchase-btn" style="margin-top: 15px; width: 100%;">Complete Purchase</button>
                    <div id="purchase-result"></div>
                </div>
            </div>
        </div>

        <div class="card">
            <h2>📋 Order History</h2>
            <div id="orders-section">
                <button class="btn btn-secondary" onclick="loadOrders()" style="margin-bottom: 15px;">Refresh Orders</button>
                <div id="orders-list"></div>
            </div>
        </div>

        <div class="card">
            <h2>Query Executor</h2>
            <div class="query-section">
                <textarea class="query-input" id="query-input" placeholder="Enter SQL query here...&#10;Example: SELECT 1 as test"></textarea>
                <button class="btn" onclick="executeQuery()">Execute Query</button>
                <button class="btn btn-secondary" onclick="clearQuery()">Clear</button>
            </div>
            <div id="query-results"></div>
        </div>
    </div>

    <script>
        // Get token from localStorage (must be outside IIFE to be accessible)
        const token = localStorage.getItem('token');
        
        // Check authentication - verify token before loading page content
        (function() {
            if (!token) {
                // No token - redirect immediately
                localStorage.clear();
                sessionStorage.clear();
                window.location.replace('/login');
                return; // Stop execution
            }
            
            // Verify token is valid before showing content
            fetch('/api/db/stats', {
                headers: {
                    'Authorization': 'Bearer ' + token,
                    'Content-Type': 'application/json'
                }
            })
            .then(res => {
                if (res.status === 401) {
                    // Token invalid - clear and redirect
                    localStorage.clear();
                    sessionStorage.clear();
                    window.location.replace('/login');
                } else if (res.status === 200) {
                    // Token valid - continue loading page
                    // Page content will load normally
                }
            })
            .catch(err => {
                console.error('Auth check error:', err);
                // On network error, clear and redirect
                localStorage.clear();
                sessionStorage.clear();
                window.location.replace('/login');
            });
        })();

        // Display username
        const username = localStorage.getItem('username');
        if (username) {
            document.getElementById('username-display').textContent = 'Logged in as: ' + username;
        }

        // Setup auth header for all requests
        const authHeaders = {
            'Authorization': 'Bearer ' + token,
            'Content-Type': 'application/json'
        };

        // Shopping cart
        let cart = [];

        // Auto-refresh stats every 2 seconds
        setInterval(updateStats, 2000);
        updateStats();

        // Load products, orders, and wallet on page load
        loadProducts();
        loadOrders();
        loadWallet();

        function handleLogout() {
            localStorage.clear();
            sessionStorage.clear();
            window.location.replace('/login');
        }

        function updateStats() {
            fetch('/api/db/stats', {
                headers: authHeaders
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(data => {
                    if (data.error) {
                        document.getElementById('status-indicator').innerHTML = 
                            '<span class="status-indicator status-offline"></span><span>Offline</span>';
                        return;
                    }

                    document.getElementById('status-indicator').innerHTML = 
                        '<span class="status-indicator status-online"></span><span>Online</span>';
                    document.getElementById('open-connections').textContent = data.open_connections || 0;
                    document.getElementById('in-use').textContent = data.in_use || 0;
                    document.getElementById('idle').textContent = data.idle || 0;
                    document.getElementById('wait-count').textContent = data.wait_count || 0;
                })
                .catch(err => {
                    document.getElementById('status-indicator').innerHTML = 
                        '<span class="status-indicator status-offline"></span><span>Error</span>';
                });
        }

        function executeQuery() {
            const query = document.getElementById('query-input').value.trim();
            if (!query) {
                alert('Please enter a query');
                return;
            }

            const resultsDiv = document.getElementById('query-results');
            resultsDiv.innerHTML = '<div class="loading"></div> Executing query...';

            fetch('/api/db/query', {
                method: 'POST',
                headers: authHeaders,
                body: JSON.stringify({ query: query })
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(data => {
                    if (!data) return;
                if (data.error) {
                    resultsDiv.innerHTML = '<div class="error">Error: ' + data.error + '</div>';
                    return;
                }

                if (data.rows && data.rows.length > 0) {
                    let html = '<div class="success">Query executed successfully. Rows: ' + data.count + '</div>';
                    html += '<div class="results"><table><thead><tr>';
                    data.columns.forEach(col => {
                        html += '<th>' + col + '</th>';
                    });
                    html += '</tr></thead><tbody>';
                    data.rows.forEach(row => {
                        html += '<tr>';
                        data.columns.forEach(col => {
                            html += '<td>' + (row[col] !== null && row[col] !== undefined ? row[col] : 'NULL') + '</td>';
                        });
                        html += '</tr>';
                    });
                    html += '</tbody></table></div>';
                    resultsDiv.innerHTML = html;
                } else {
                    resultsDiv.innerHTML = '<div class="success">Query executed successfully. No rows returned.</div>';
                }
            })
            .catch(err => {
                resultsDiv.innerHTML = '<div class="error">Error: ' + err.message + '</div>';
            });
        }

        function clearQuery() {
            document.getElementById('query-input').value = '';
            document.getElementById('query-results').innerHTML = '';
        }

        // Allow Ctrl+Enter to execute query
        document.getElementById('query-input').addEventListener('keydown', function(e) {
            if (e.ctrlKey && e.key === 'Enter') {
                executeQuery();
            }
        });

        // Purchase functions
        function loadProducts() {
            fetch('/api/products', {
                headers: authHeaders
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(products => {
                    if (!products) return;
                    const productsList = document.getElementById('products-list');
                    productsList.innerHTML = products.map(function(product) {
                        return '<div style="background: #f8f9fa; padding: 15px; border-radius: 8px; border: 2px solid #e0e0e0;">' +
                            '<h4 style="margin: 0 0 10px 0; color: #333;">' + product.name + '</h4>' +
                            '<p style="margin: 0 0 10px 0; color: #666; font-size: 0.9em;">' + (product.description || '') + '</p>' +
                            '<div style="display: flex; justify-content: space-between; align-items: center; margin-top: 10px;">' +
                            '<div>' +
                            '<strong style="color: #667eea; font-size: 1.2em;">$' + product.price.toFixed(2) + '</strong>' +
                            '<div style="font-size: 0.85em; color: #666;">Stock: ' + product.stock + '</div>' +
                            '</div>' +
                            '<div>' +
                            '<input type="number" id="qty-' + product.id + '" min="1" max="' + product.stock + '" value="1" ' +
                            'style="width: 60px; padding: 5px; margin-right: 5px; border: 2px solid #e0e0e0; border-radius: 4px;">' +
                            '<button class="btn" onclick="addToCart(' + product.id + ', \'' + product.name.replace(/'/g, "\\'") + '\', ' + product.price + ', ' + product.stock + ')" ' +
                            'style="padding: 5px 15px; font-size: 0.9em;">Add</button>' +
                            '</div>' +
                            '</div>' +
                            '</div>';
                    }).join('');
                })
                .catch(err => {
                    console.error('Failed to load products:', err);
                });
        }

        function addToCart(productId, name, price, stock) {
            const qtyInput = document.getElementById('qty-' + productId);
            const quantity = parseInt(qtyInput.value) || 1;
            
            if (quantity > stock) {
                alert('Insufficient stock!');
                return;
            }

            const existingItem = cart.find(item => item.product_id === productId);
            if (existingItem) {
                existingItem.quantity += quantity;
            } else {
                cart.push({
                    product_id: productId,
                    name: name,
                    price: price,
                    quantity: quantity
                });
            }

            updateCartDisplay();
        }

        function updateCartDisplay() {
            const cartItems = document.getElementById('cart-items');
            const cartTotal = document.getElementById('cart-total');
            
            if (cart.length === 0) {
                cartItems.innerHTML = '<p style="color: #666;">Cart is empty</p>';
                cartTotal.textContent = '';
                return;
            }

            let total = 0;
            cartItems.innerHTML = cart.map(function(item, index) {
                const itemTotal = item.price * item.quantity;
                total += itemTotal;
                return '<div style="display: flex; justify-content: space-between; align-items: center; padding: 10px; background: #f8f9fa; border-radius: 8px; margin-bottom: 10px;">' +
                    '<div>' +
                    '<strong>' + item.name + '</strong>' +
                    '<div style="font-size: 0.9em; color: #666;">$' + item.price.toFixed(2) + ' x ' + item.quantity + ' = $' + itemTotal.toFixed(2) + '</div>' +
                    '</div>' +
                    '<button class="btn btn-secondary" onclick="removeFromCart(' + index + ')" style="padding: 5px 10px; font-size: 0.85em;">Remove</button>' +
                    '</div>';
            }).join('');

            cartTotal.textContent = 'Total: $' + total.toFixed(2);
        }

        function removeFromCart(index) {
            cart.splice(index, 1);
            updateCartDisplay();
        }

        function makePurchase() {
            if (cart.length === 0) {
                alert('Cart is empty!');
                return;
            }

            const purchaseBtn = document.getElementById('purchase-btn');
            const resultDiv = document.getElementById('purchase-result');
            purchaseBtn.disabled = true;
            purchaseBtn.textContent = 'Processing...';
            resultDiv.innerHTML = '';

            const purchaseData = {
                // user_id removed - server will get it from JWT token
                items: cart.map(item => ({
                    product_id: item.product_id,
                    quantity: item.quantity
                }))
            };

            fetch('/api/purchase', {
                method: 'POST',
                headers: authHeaders,
                body: JSON.stringify(purchaseData)
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(data => {
                    if (!data) return;
                    if (data.error) {
                        resultDiv.innerHTML = '<div class="error">Error: ' + data.message + '</div>';
                    } else {
                        resultDiv.innerHTML = '<div class="success">Purchase successful! Order ID: ' + data.order_id + ', Total: $' + data.total.toFixed(2) + '</div>';
                        cart = [];
                        updateCartDisplay();
                        loadProducts();
                        loadOrders();
                        loadWallet(); // Reload wallet to show updated balance
                    }
                    purchaseBtn.disabled = false;
                    purchaseBtn.textContent = 'Complete Purchase';
                })
                .catch(err => {
                    resultDiv.innerHTML = '<div class="error">Error: ' + err.message + '</div>';
                    purchaseBtn.disabled = false;
                    purchaseBtn.textContent = 'Complete Purchase';
                });
        }

        function loadOrders() {
            // Use the logged-in username for orders
            const orderUserID = username || 'demo_user';
            fetch('/api/orders?user_id=' + encodeURIComponent(orderUserID), {
                headers: authHeaders
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(orders => {
                    if (!orders) return;
                    const ordersList = document.getElementById('orders-list');
                    if (orders.length === 0) {
                        ordersList.innerHTML = '<p style="color: #666;">No orders yet</p>';
                        return;
                    }
                    ordersList.innerHTML = orders.map(function(order) {
                        const date = new Date(order.created_at).toLocaleString();
                        const itemsHtml = order.items.map(function(item) {
                            return '<div style="font-size: 0.9em; color: #666; margin-bottom: 5px;">' +
                                'Product #' + item.product_id + ' - Qty: ' + item.quantity + ' - $' + item.subtotal.toFixed(2) +
                                '</div>';
                        }).join('');
                        return '<div style="background: #f8f9fa; padding: 15px; border-radius: 8px; margin-bottom: 15px; border-left: 4px solid #667eea;">' +
                            '<div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: 10px;">' +
                            '<div>' +
                            '<strong>Order #' + order.id + '</strong>' +
                            '<div style="font-size: 0.9em; color: #666;">' + date + '</div>' +
                            '</div>' +
                            '<div style="text-align: right;">' +
                            '<div style="font-size: 1.2em; font-weight: bold; color: #667eea;">$' + order.total.toFixed(2) + '</div>' +
                            '<div style="font-size: 0.85em; color: #666;">Status: ' + order.status + '</div>' +
                            '</div>' +
                            '</div>' +
                            '<div style="margin-top: 10px; padding-top: 10px; border-top: 1px solid #dee2e6;">' +
                            itemsHtml +
                            '</div>' +
                            '</div>';
                    }).join('');
                })
                .catch(err => {
                    console.error('Failed to load orders:', err);
                });
        }

        function loadWallet() {
            // Load wallet balance
            fetch('/api/wallet/balance', {
                headers: authHeaders
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(data => {
                    if (!data) return;
                    const balanceEl = document.getElementById('wallet-balance');
                    const totalEl = document.getElementById('wallet-total');
                    const frozenEl = document.getElementById('wallet-frozen');
                    
                    if (balanceEl) {
                        // Show available balance (balance - frozen)
                        const available = (data.available_balance !== undefined) ? data.available_balance : 
                                         ((data.balance || 0) - (data.frozen || 0));
                        balanceEl.textContent = '$' + available.toFixed(2);
                    }
                    if (totalEl) {
                        totalEl.textContent = (data.balance || 0).toFixed(2);
                    }
                    if (frozenEl) {
                        frozenEl.textContent = (data.frozen || 0).toFixed(2);
                    }
                })
                .catch(err => {
                    console.error('Failed to load wallet balance:', err);
                });

            // Load wallet transactions
            fetch('/api/wallet/transactions', {
                headers: authHeaders
            })
                .then(res => {
                    if (res.status === 401) {
                        handleLogout();
                        return;
                    }
                    return res.json();
                })
                .then(transactions => {
                    if (!transactions) return;
                    const transactionsEl = document.getElementById('wallet-transactions');
                    if (!transactionsEl) return;

                    if (transactions.length === 0) {
                        transactionsEl.innerHTML = '<p style="color: #666; text-align: center;">No transactions yet</p>';
                        return;
                    }

                    transactionsEl.innerHTML = transactions.map(function(t) {
                        const date = new Date(t.created_at).toLocaleString();
                        const typeColor = t.type === 'debit' ? '#dc3545' : '#28a745';
                        const typeIcon = t.type === 'debit' ? '−' : '+';
                        const orderLink = t.order_id ? ' (Order #' + t.order_id + ')' : '';
                        return '<div style="padding: 10px; margin-bottom: 8px; background: white; border-radius: 6px; border-left: 3px solid ' + typeColor + ';">' +
                            '<div style="display: flex; justify-content: space-between; align-items: center;">' +
                            '<div>' +
                            '<div style="font-weight: bold; color: ' + typeColor + ';">' + typeIcon + ' $' + t.amount.toFixed(2) + '</div>' +
                            '<div style="font-size: 0.85em; color: #666;">' + (t.description || '') + orderLink + '</div>' +
                            '</div>' +
                            '<div style="font-size: 0.8em; color: #999;">' + date + '</div>' +
                            '</div>' +
                            '</div>';
                    }).join('');
                })
                .catch(err => {
                    console.error('Failed to load wallet transactions:', err);
                });
        }

        // Add balance button handler
        document.addEventListener('DOMContentLoaded', function() {
            const addBalanceBtn = document.getElementById('add-balance-btn');
            if (addBalanceBtn) {
                addBalanceBtn.addEventListener('click', function() {
                    fetch('/api/wallet/add', {
                        method: 'POST',
                        headers: authHeaders,
                        body: JSON.stringify({
                            amount: 100,
                            description: 'Wallet top-up'
                        })
                    })
                        .then(res => {
                            if (res.status === 401) {
                                handleLogout();
                                return;
                            }
                            return res.json();
                        })
                        .then(data => {
                            if (!data) return;
                            if (data.error) {
                                alert('Error: ' + data.message);
                            } else {
                                loadWallet();
                                alert('$100 added to wallet! New balance: $' + data.balance.toFixed(2));
                            }
                        })
                        .catch(err => {
                            console.error('Failed to add balance:', err);
                            alert('Error adding balance');
                        });
                });
            }
        });
    </script>
</body>
</html>`
}
