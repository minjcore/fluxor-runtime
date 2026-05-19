package devops

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunCertbotOptions holds options for running certbot on the VPS.
type RunCertbotOptions struct {
	Force bool // Skip confirmation
}

// RunCertbot runs certbot --nginx on the VPS to obtain/renew SSL for the target's domains.
// Email: from target.Certbot.Email, or CERTBOT_EMAIL in .env.local / env.
// Requires certbot and python3-certbot-nginx on the VPS (apt install certbot python3-certbot-nginx).
func RunCertbot(target DeployTarget, opts RunCertbotOptions) error {
	if len(target.Certbot.Domains) == 0 {
		return fmt.Errorf("certbot domains required: add certbot.domains to target in deploy.yaml (e.g. [quadgate.io, www.quadgate.io])")
	}

	email := target.Certbot.Email
	if email == "" {
		envVars, _ := loadEnvFile("")
		if e, ok := envVars["CERTBOT_EMAIL"]; ok && e != "" {
			email = e
		}
		if email == "" {
			email = os.Getenv("CERTBOT_EMAIL")
		}
	}
	if email == "" {
		return fmt.Errorf("certbot email required: add CERTBOT_EMAIL=your@email.com to .env.local (project root), or set certbot.email in deploy.yaml")
	}

	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Build: certbot --nginx --non-interactive --agree-tos --email X -d domain1 -d domain2 ...
	args := []string{"--nginx", "--non-interactive", "--agree-tos", "--email", email}
	for _, d := range target.Certbot.Domains {
		d = strings.TrimSpace(d)
		if d != "" {
			args = append(args, "-d", d)
		}
	}
	cmd := "certbot " + strings.Join(args, " ")

	if !opts.Force {
		fmt.Printf("This will run on %s: %s\n", target.Host, cmd)
		fmt.Printf("Continue? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("certbot cancelled by user")
		}
	}

	fmt.Printf("Running certbot on VPS...\n")
	err = client.ExecuteCommandWithSudo(cmd)
	if err != nil {
		errStr := err.Error()
		// Cert was obtained but nginx restart failed (e.g. address already in use)
		if strings.Contains(errStr, "Successfully deployed certificate") &&
			(strings.Contains(errStr, "nginx restart failed") || strings.Contains(errStr, "Address already in use")) {
			fmt.Printf("✅ Certificate obtained and deployed to nginx config. Nginx restart failed (port conflict).\n")
			fmt.Printf("   On VPS run: sudo systemctl stop nginx && sleep 2 && sudo systemctl start nginx\n")
			fmt.Printf("   Or check: sudo nginx -t && sudo systemctl restart nginx\n")
			return nil
		}
		return fmt.Errorf("certbot failed: %w (on VPS ensure certbot is installed: apt install certbot python3-certbot-nginx)", err)
	}
	fmt.Printf("✅ Certbot completed. SSL certificates installed for %s\n", strings.Join(target.Certbot.Domains, ", "))
	return nil
}
