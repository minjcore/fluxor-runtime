-- ============================================================================
-- UPDATE SYSTEM SETTINGS TABLE
-- Adds indexes, constraints, triggers, and default data
-- Note: Table is already created in 000_initial_setup.sql
-- ============================================================================

-- Create indexes for efficient lookup
CREATE INDEX IF NOT EXISTS idx_system_settings_key ON system_settings(setting_key);
CREATE INDEX IF NOT EXISTS idx_system_settings_category ON system_settings(category);
CREATE INDEX IF NOT EXISTS idx_system_settings_readonly ON system_settings(is_readonly);

-- Add CHECK constraints (idempotent)
DO $$ 
BEGIN
    -- Add data_type constraint if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'system_settings_data_type_check'
    ) THEN
        ALTER TABLE system_settings 
        ADD CONSTRAINT system_settings_data_type_check 
        CHECK (data_type IN ('string', 'integer', 'float', 'boolean', 'json', 'array', 'object'));
    END IF;

    -- Add category constraint if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'system_settings_category_check'
    ) THEN
        ALTER TABLE system_settings 
        ADD CONSTRAINT system_settings_category_check 
        CHECK (category IS NULL OR LENGTH(category) > 0);
    END IF;
END $$;

-- Create trigger for automatic updated_at timestamp
DROP TRIGGER IF EXISTS trigger_system_settings_updated_at ON system_settings;
CREATE TRIGGER trigger_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INSERT DEFAULT SYSTEM SETTINGS
-- ============================================================================

-- Wallet Settings
INSERT INTO system_settings (setting_key, setting_value, data_type, category, description, is_readonly, default_value, metadata) VALUES
    ('wallet.auto_create_on_registration', 'true', 'boolean', 'wallet', 'Automatically create wallets for new users', false, 'true', '{}'::jsonb),
    ('wallet.default_initial_balance', '0.00', 'float', 'wallet', 'Default initial balance for new wallets', false, '0.00', '{"min": 0}'::jsonb),
    ('wallet.max_wallets_per_user', '10', 'integer', 'wallet', 'Maximum number of wallets a user can have', false, '10', '{"min": 1, "max": 100}'::jsonb),
    ('wallet.transaction_fee_percentage', '0.00', 'float', 'wallet', 'Transaction fee as percentage', false, '0.00', '{"min": 0, "max": 10}'::jsonb),
    ('wallet.min_transfer_amount', '0.01', 'float', 'wallet', 'Minimum transfer amount', false, '0.01', '{"min": 0}'::jsonb),
    ('wallet.max_transfer_amount', '100000.00', 'float', 'wallet', 'Maximum transfer amount per transaction', false, '100000.00', '{"min": 0}'::jsonb),

-- Security Settings
    ('security.jwt_expiry_hours', '24', 'integer', 'security', 'JWT token expiry time in hours', false, '24', '{"min": 1, "max": 720}'::jsonb),
    ('security.password_min_length', '8', 'integer', 'security', 'Minimum password length', false, '8', '{"min": 6, "max": 128}'::jsonb),
    ('security.max_login_attempts', '5', 'integer', 'security', 'Maximum failed login attempts before lockout', false, '5', '{"min": 3, "max": 10}'::jsonb),
    ('security.lockout_duration_minutes', '30', 'integer', 'security', 'Account lockout duration in minutes', false, '30', '{"min": 5, "max": 1440}'::jsonb),
    ('security.require_2fa', 'false', 'boolean', 'security', 'Require two-factor authentication', false, 'false', '{}'::jsonb),

-- Application Settings
    ('app.name', 'Fluxor Postgres Demo', 'string', 'application', 'Application name', false, 'Fluxor Postgres Demo', '{}'::jsonb),
    ('app.version', '1.0.0', 'string', 'application', 'Application version', true, '1.0.0', '{}'::jsonb),
    ('app.maintenance_mode', 'false', 'boolean', 'application', 'Enable maintenance mode', false, 'false', '{}'::jsonb),
    ('app.maintenance_message', 'System is under maintenance. Please try again later.', 'string', 'application', 'Maintenance mode message', false, 'System is under maintenance. Please try again later.', '{}'::jsonb),
    ('app.max_request_size_mb', '10', 'integer', 'application', 'Maximum request size in MB', false, '10', '{"min": 1, "max": 100}'::jsonb),
    ('app.session_timeout_minutes', '60', 'integer', 'application', 'Session timeout in minutes', false, '60', '{"min": 5, "max": 1440}'::jsonb),

-- Notification Settings
    ('notification.email_enabled', 'false', 'boolean', 'notification', 'Enable email notifications', false, 'false', '{}'::jsonb),
    ('notification.sms_enabled', 'false', 'boolean', 'notification', 'Enable SMS notifications', false, 'false', '{}'::jsonb),
    ('notification.push_enabled', 'true', 'boolean', 'notification', 'Enable push notifications', false, 'true', '{}'::jsonb),
    ('notification.transaction_alerts', 'true', 'boolean', 'notification', 'Send alerts for transactions', false, 'true', '{}'::jsonb),

-- Database Settings
    ('database.max_connections', '25', 'integer', 'database', 'Maximum database connections', false, '25', '{"min": 5, "max": 1000}'::jsonb),
    ('database.connection_timeout_seconds', '30', 'integer', 'database', 'Database connection timeout', false, '30', '{"min": 5, "max": 300}'::jsonb),
    ('database.query_timeout_seconds', '60', 'integer', 'database', 'Query timeout in seconds', false, '60', '{"min": 5, "max": 600}'::jsonb),

-- Feature Flags
    ('feature.ledger_enabled', 'true', 'boolean', 'feature', 'Enable ledger/double-entry bookkeeping', false, 'true', '{}'::jsonb),
    ('feature.wallet_rules_enabled', 'true', 'boolean', 'feature', 'Enable wallet rules validation', false, 'true', '{}'::jsonb),
    ('feature.accountant_dashboard_enabled', 'true', 'boolean', 'feature', 'Enable accountant dashboard', false, 'true', '{}'::jsonb),
    ('feature.transfer_between_wallets', 'true', 'boolean', 'feature', 'Enable transfers between user wallets', false, 'true', '{}'::jsonb)
ON CONFLICT (setting_key) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE system_settings IS 'System-wide configuration settings stored as key-value pairs';
COMMENT ON COLUMN system_settings.setting_key IS 'Unique setting identifier (e.g., wallet.auto_create_on_registration)';
COMMENT ON COLUMN system_settings.setting_value IS 'Setting value stored as text (parsed according to data_type)';
COMMENT ON COLUMN system_settings.data_type IS 'Data type: string, integer, float, boolean, json, array, object';
COMMENT ON COLUMN system_settings.category IS 'Category/group for organizing settings (e.g., wallet, security, notification)';
COMMENT ON COLUMN system_settings.is_encrypted IS 'Whether the value is encrypted (for sensitive data like API keys)';
COMMENT ON COLUMN system_settings.is_readonly IS 'Whether this setting can be modified via API (false = can be modified)';
COMMENT ON COLUMN system_settings.validation_rule IS 'JSON validation rule (e.g., {"min": 0, "max": 100, "pattern": "^[A-Z]+$"})';
COMMENT ON COLUMN system_settings.default_value IS 'Default value if setting is not set or reset';
COMMENT ON COLUMN system_settings.metadata IS 'Additional metadata in JSON format';
