package views

// GetRegisterHTML returns the HTML for the registration page
func GetRegisterHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PostgreSQL Demo - Register</title>
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
        .register-container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            width: 100%;
            max-width: 450px;
        }
        .register-container h1 {
            color: #333;
            margin-bottom: 10px;
            text-align: center;
        }
        .register-container p {
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
        .btn-secondary {
            background: #6c757d;
            margin-top: 10px;
        }
        .btn-secondary:hover {
            background: #5a6268;
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
        .success {
            background: #d4edda;
            color: #155724;
            padding: 12px;
            border-radius: 8px;
            margin-top: 15px;
            display: none;
        }
        .success.show {
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
        .password-requirements {
            background: #e7f3ff;
            border-left: 4px solid #667eea;
            padding: 12px;
            border-radius: 8px;
            margin-top: 10px;
            font-size: 13px;
            color: #666;
        }
        .password-requirements ul {
            margin: 8px 0 0 20px;
        }
        .login-link {
            text-align: center;
            margin-top: 20px;
            color: #666;
            font-size: 14px;
        }
        .login-link a {
            color: #667eea;
            text-decoration: none;
            font-weight: 500;
        }
        .login-link a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="register-container">
        <h1>🐘 PostgreSQL Demo</h1>
        <p>Create a new account</p>
        <form id="register-form" onsubmit="handleRegister(event)">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus minlength="3" maxlength="50">
            </div>
            <div class="form-group">
                <label for="email">Email (Optional)</label>
                <input type="email" id="email" name="email" maxlength="255">
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required minlength="6">
                <div class="password-requirements">
                    <strong>Password requirements:</strong>
                    <ul>
                        <li>At least 6 characters</li>
                        <li>Recommended: mix of letters, numbers, and symbols</li>
                    </ul>
                </div>
            </div>
            <div class="form-group">
                <label for="confirm-password">Confirm Password</label>
                <input type="password" id="confirm-password" name="confirm-password" required>
            </div>
            <button type="submit" class="btn" id="register-btn">
                Register
                <span class="loading" id="loading"></span>
            </button>
            <div class="error" id="error"></div>
            <div class="success" id="success"></div>
        </form>
        <div class="login-link">
            Already have an account? <a href="/login">Login here</a>
        </div>
    </div>

    <script>
        function handleRegister(event) {
            event.preventDefault();
            
            const username = document.getElementById('username').value.trim();
            const email = document.getElementById('email').value.trim();
            const password = document.getElementById('password').value;
            const confirmPassword = document.getElementById('confirm-password').value;
            const errorDiv = document.getElementById('error');
            const successDiv = document.getElementById('success');
            const loading = document.getElementById('loading');
            const registerBtn = document.getElementById('register-btn');
            
            errorDiv.classList.remove('show');
            successDiv.classList.remove('show');
            
            // Validate passwords match
            if (password !== confirmPassword) {
                errorDiv.textContent = 'Passwords do not match';
                errorDiv.classList.add('show');
                return;
            }
            
            // Validate password length
            if (password.length < 6) {
                errorDiv.textContent = 'Password must be at least 6 characters';
                errorDiv.classList.add('show');
                return;
            }
            
            loading.classList.add('show');
            registerBtn.disabled = true;

            // Build request payload
            const payload = {
                username: username,
                password: password
            };
            
            // Add email if provided
            if (email) {
                payload.email = email;
            }

            fetch('/api/auth/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(payload)
            })
            .then(res => res.json())
            .then(data => {
                loading.classList.remove('show');
                registerBtn.disabled = false;

                if (data.error) {
                    errorDiv.textContent = data.message || 'Registration failed';
                    errorDiv.classList.add('show');
                    return;
                }

                // Show success message
                successDiv.textContent = 'Registration successful! Redirecting to login...';
                successDiv.classList.add('show');
                
                // Redirect to login page after 2 seconds
                setTimeout(() => {
                    window.location.replace('/login?registered=true');
                }, 2000);
            })
            .catch(err => {
                loading.classList.remove('show');
                registerBtn.disabled = false;
                errorDiv.textContent = 'Network error: ' + err.message;
                errorDiv.classList.add('show');
            });
        }
        
        // Check if redirected from successful registration
        const urlParams = new URLSearchParams(window.location.search);
        if (urlParams.get('registered') === 'true') {
            const successDiv = document.getElementById('success');
            successDiv.textContent = 'Registration successful! Please login with your credentials.';
            successDiv.classList.add('show');
        }
    </script>
</body>
</html>`
}
