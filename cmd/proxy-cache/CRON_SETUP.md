# Scheduled Vulnerability Scanning with Cron

This guide explains how to set up automated vulnerability scanning using `govulncheck` via cron jobs.

## Quick Start

### Install Cron Job (Daily at 2 AM)
```bash
make cron-install
```

### Remove Cron Job
```bash
make cron-remove
```

### View Current Cron Jobs
```bash
make cron-list
```

### Manual Scan
```bash
./scan-vulns.sh
```

## Manual Cron Setup

If you prefer to set up the cron job manually, add this line to your crontab:

```bash
crontab -e
```

Then add one of the following schedules:

### Daily at 2:00 AM (Recommended)
```
0 2 * * * /path/to/proxy-cache/scan-vulns.sh > /dev/null 2>&1
```

### Every 6 Hours
```
0 */6 * * * /path/to/proxy-cache/scan-vulns.sh > /dev/null 2>&1
```

### Every Sunday at Midnight
```
0 0 * * 0 /path/to/proxy-cache/scan-vulns.sh > /dev/null 2>&1
```

### Weekly on Monday at 3 AM
```
0 3 * * 1 /path/to/proxy-cache/scan-vulns.sh > /dev/null 2>&1
```

## Cron Schedule Format

The cron schedule format is:
```
* * * * *
│ │ │ │ │
│ │ │ │ └─── Day of week (0-7, Sunday = 0 or 7)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

Examples:
- `0 2 * * *` - Daily at 2:00 AM
- `0 */6 * * *` - Every 6 hours
- `30 14 * * 1-5` - Every weekday at 2:30 PM
- `0 0 1 * *` - First day of every month at midnight

## Log Files

Scan results are logged to:
- **Log directory**: `logs/vulnscan-YYYYMMDD-HHMMSS.log`
- **Error log**: `logs/vulnscan-errors.log`

Logs are automatically rotated, keeping the last 30 days of logs.

## Viewing Logs

```bash
# View latest log
ls -t logs/vulnscan-*.log | head -1 | xargs cat

# View all logs
ls -lt logs/vulnscan-*.log

# View error log
cat logs/vulnscan-errors.log

# Tail latest log in real-time (if running manually)
tail -f logs/vulnscan-*.log
```

## Email Notifications

To receive email notifications when vulnerabilities are found, modify the cron job:

```bash
0 2 * * * /path/to/proxy-cache/scan-vulns.sh | mail -s "Vulnerability Scan Results" your-email@example.com
```

Or configure email in the script itself (see script comments).

## Troubleshooting

### Script Not Found
If cron reports "script not found", ensure:
1. Use absolute path in crontab
2. Script has execute permissions: `chmod +x scan-vulns.sh`

### Go Environment Not Found
The script automatically sets up the Go environment, but if issues persist:
1. Ensure `go` is in cron's PATH
2. Check that `GOPATH` is set correctly
3. Verify `govulncheck` is installed: `go install golang.org/x/vuln/cmd/govulncheck@latest`

### Check Cron Service
```bash
# macOS (using launchd instead of cron)
launchctl list | grep cron

# Linux
systemctl status cron
# or
service cron status
```

### View Cron Logs
```bash
# macOS - Check system logs
log show --predicate 'process == "cron"' --last 1h

# Linux - Check syslog
grep CRON /var/log/syslog

# Check mail for cron output (if configured)
mail
```

## Security Best Practices

1. **Regular Scanning**: Run scans at least weekly, preferably daily
2. **Monitor Logs**: Set up log monitoring/alerting for vulnerability findings
3. **Update Dependencies**: After finding vulnerabilities, update affected packages
4. **CI Integration**: Also run scans in CI/CD pipelines
5. **Review Findings**: Don't ignore warnings; review and address security issues promptly

## Integration with CI/CD

For continuous integration, consider adding to your CI pipeline:

```yaml
# Example GitHub Actions
- name: Run vulnerability scan
  run: |
    cd cmd/proxy-cache
    make govulncheck
```

```bash
# Example GitLab CI
vulnerability_scan:
  script:
    - cd cmd/proxy-cache
    - make govulncheck
```
