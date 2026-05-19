package devops

// DeployConfig represents the deployment configuration loaded from YAML file.
type DeployConfig struct {
	Targets map[string]DeployTarget `yaml:"targets"`
}

// DeployTarget represents a deployment target configuration.
type DeployTarget struct {
	Host         string              `yaml:"host"`
	User         string              `yaml:"user"`
	SSHKey       string              `yaml:"ssh_key"`
	AppDir       string              `yaml:"app_dir"` // /opt/fluxor
	GoApp        GoAppConfig         `yaml:"go_app"`
	NodeApp      NodeAppConfig       `yaml:"node_app"`
	Nginx        NginxConfig         `yaml:"nginx"`
	Certbot      CertbotConfig       `yaml:"certbot"`
	DockerCompose DockerComposeConfig `yaml:"docker_compose"`
}

// CertbotConfig is optional; used by deploy -certbot to obtain SSL via Let's Encrypt.
type CertbotConfig struct {
	Domains []string `yaml:"domains"` // e.g. [quadgate.io, www.quadgate.io]
	Email   string   `yaml:"email"`   // for Let's Encrypt; can override with CERTBOT_EMAIL in .env.local
}

// GoAppConfig represents Go application configuration for a deployment target.
type GoAppConfig struct {
	Source      string            `yaml:"source"`       // cmd/ssr-example (local source directory)
	ServiceName string            `yaml:"service_name"` // fluxor-ssr-go
	BinaryPath  string            `yaml:"binary_path"`  // /opt/fluxor/ssr-app/ssr-app-linux-amd64
	BinaryName  string            `yaml:"binary_name"` // ssr-app-linux-amd64 (binary filename)
	Env         map[string]string `yaml:"env"`         // Environment variables for systemd service
}

// NodeAppConfig represents Node.js application configuration for a deployment target.
type NodeAppConfig struct {
	Source      string `yaml:"source"`       // examples/ssr-app (local source directory)
	BuildOutput string `yaml:"build_output"` // dist/client (build output directory)
	DestDir     string `yaml:"dest_dir"`     // /opt/fluxor/ssr-app/dist/client (destination on VPS)
}

// NginxConfig represents nginx configuration for a deployment target.
type NginxConfig struct {
	ConfigSource string `yaml:"config_source"` // ssrflux.com.conf (local nginx config file)
	ConfigDest   string `yaml:"config_dest"`  // /etc/nginx/sites-available/ssrflux.com
	EnabledPath  string `yaml:"enabled_path"` // /etc/nginx/sites-enabled/ssrflux.com
	SiteName     string `yaml:"site_name"`    // ssrflux.com
}

// DockerComposeConfig represents Docker Compose configuration for a deployment target.
type DockerComposeConfig struct {
	ComposeFile  string `yaml:"compose_file"`  // wordpress.yaml (local docker-compose file)
	RemoteDir    string `yaml:"remote_dir"`   // /opt/wordpress (destination directory on VPS)
	ProjectName  string `yaml:"project_name"` // wordpress (docker-compose project name)
}

// DefaultDockerComposeConfig returns default Docker Compose config. Override ComposeFile, RemoteDir, ProjectName for your app.
func DefaultDockerComposeConfig() DockerComposeConfig {
	return DockerComposeConfig{
		ComposeFile:  "docker-compose.yml",
		RemoteDir:    "/opt/app",
		ProjectName:  "app",
	}
}
