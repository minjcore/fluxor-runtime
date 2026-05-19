-- Fluxor Enterprise Database Seed Data
-- This file contains realistic seed data for development and testing

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] NOT NULL DEFAULT ARRAY['user'],
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on email for fast lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Insert seed users
INSERT INTO users (id, email, name, password_hash, roles) VALUES
    ('user-001', 'admin@example.com', 'Admin User', '$2a$10$N9qo8uLOickgx2ZMRZoMye/p6jXP.QU8K9hGNP9DSfT5mWYiTsyy2', ARRAY['admin', 'user']),
    ('user-002', 'john.doe@example.com', 'John Doe', '$2a$10$N9qo8uLOickgx2ZMRZoMye/p6jXP.QU8K9hGNP9DSfT5mWYiTsyy2', ARRAY['user']),
    ('user-003', 'jane.smith@example.com', 'Jane Smith', '$2a$10$N9qo8uLOickgx2ZMRZoMye/p6jXP.QU8K9hGNP9DSfT5mWYiTsyy2', ARRAY['user']),
    ('user-004', 'bob.wilson@example.com', 'Bob Wilson', '$2a$10$N9qo8uLOickgx2ZMRZoMye/p6jXP.QU8K9hGNP9DSfT5mWYiTsyy2', ARRAY['user'])
ON CONFLICT (id) DO NOTHING;

-- Create products table (example)
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert seed products
INSERT INTO products (id, name, description, price, stock) VALUES
    ('prod-001', 'Laptop', 'High-performance laptop', 999.99, 50),
    ('prod-002', 'Mouse', 'Wireless mouse', 29.99, 200),
    ('prod-003', 'Keyboard', 'Mechanical keyboard', 79.99, 150),
    ('prod-004', 'Monitor', '27-inch 4K monitor', 399.99, 75),
    ('prod-005', 'Headphones', 'Noise-cancelling headphones', 149.99, 100)
ON CONFLICT (id) DO NOTHING;

-- Create orders table (example)
CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    total DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on user_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);

-- Insert seed orders
INSERT INTO orders (id, user_id, total, status) VALUES
    ('order-001', 'user-002', 1029.98, 'completed'),
    ('order-002', 'user-003', 79.99, 'pending'),
    ('order-003', 'user-002', 149.99, 'shipped')
ON CONFLICT (id) DO NOTHING;

-- Note: Password hash above is bcrypt hash of 'password' (for development only)
-- In production, users should create their own secure passwords
