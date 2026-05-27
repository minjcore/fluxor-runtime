package views

// GetLoginHTML returns the HTML for the login page
func GetLoginHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PostgreSQL Demo - Login</title>
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
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .login-container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            width: 100%;
            max-width: 400px;
        }
        .login-container h1 {
            color: #333;
            margin-bottom: 10px;
            text-align: center;
        }
        .login-container p {
            color: #666;
            text-align: center;
            margin-bottom: 30px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        .form-group label {
            display: block;
            color: #333;
            margin-bottom: 8px;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 16px;
            transition: border-color 0.3s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn {
            width: 100%;
            background: #667eea;
            color: white;
            border: none;
            padding: 12px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            transition: all 0.3s;
            margin-top: 10px;
        }
        .btn:hover {
            background: #5568d3;
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        .btn:active {
            transform: translateY(0);
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 12px;
            border-radius: 8px;
            margin-top: 15px;
            display: none;
        }
        .error.show {
            display: block;
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
            display: none;
        }
        .loading.show {
            display: inline-block;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .demo-credentials {
            background: #e7f3ff;
            border-left: 4px solid #667eea;
            padding: 12px;
            border-radius: 8px;
            margin-top: 20px;
            font-size: 14px;
        }
        .demo-credentials strong {
            color: #333;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>🐘 PostgreSQL Demo</h1>
        <p>Please login to continue</p>
        <form id="login-form" onsubmit="handleLogin(event)">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn" id="login-btn">
                Login
                <span class="loading" id="loading"></span>
            </button>
            <div class="error" id="error"></div>
        </form>
        <div class="demo-credentials">
            <strong>Demo Credentials:</strong><br>
            Username: <code>khangdc</code><br>
            Password: <code>admin123</code>
        </div>
        <div style="text-align: center; margin-top: 20px; color: #666; font-size: 14px;">
            Don't have an account? <a href="/register" style="color: #667eea; text-decoration: none; font-weight: 500;">Register here</a>
        </div>
    </div>

    <script>
        // Check if already logged in - verify token is valid first
        // Use a flag to prevent multiple redirects
        (function() {
            // Check if we've already verified (prevent loops)
            if (sessionStorage.getItem('login_verified')) {
                return;
            }
            
            const token = localStorage.getItem('token');
            if (!token) {
                return; // No token, stay on login page
            }
            
            // Mark as verifying to prevent loops
            sessionStorage.setItem('login_verified', 'true');
            
            // Verify token by making a test request
            fetch('/api/db/stats', {
                headers: {
                    'Authorization': 'Bearer ' + token,
                    'Content-Type': 'application/json'
                }
            })
            .then(res => {
                if (res.status === 200) {
                    // Token is valid - redirect to dashboard (only once)
                    window.location.replace('/dashboard');
                } else if (res.status === 401) {
                    // Token is invalid - clear it and stay on login
                    localStorage.removeItem('token');
                    localStorage.removeItem('username');
                    sessionStorage.removeItem('login_verified');
                }
            })
            .catch(err => {
                // On error, clear token and stay on login page
                console.error('Token verification error:', err);
                localStorage.removeItem('token');
                localStorage.removeItem('username');
                sessionStorage.removeItem('login_verified');
            });
        })();

        function handleLogin(event) {
            event.preventDefault();
            
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const errorDiv = document.getElementById('error');
            const loading = document.getElementById('loading');
            const loginBtn = document.getElementById('login-btn');
            
            errorDiv.classList.remove('show');
            loading.classList.add('show');
            loginBtn.disabled = true;

            fetch('/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password })
            })
            .then(res => res.json())
            .then(data => {
                loading.classList.remove('show');
                loginBtn.disabled = false;

                if (data.error) {
                    errorDiv.textContent = data.message || 'Login failed';
                    errorDiv.classList.add('show');
                    return;
                }

                // Store token
                localStorage.setItem('token', data.token);
                localStorage.setItem('username', data.username);
                
                // Clear verification flag and redirect
                sessionStorage.removeItem('login_verified');
                window.location.replace('/dashboard');
            })
            .catch(err => {
                loading.classList.remove('show');
                loginBtn.disabled = false;
                errorDiv.textContent = 'Network error: ' + err.message;
                errorDiv.classList.add('show');
            });
        }
    </script>
</body>
</html>`
}
