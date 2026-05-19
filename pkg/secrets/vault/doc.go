// Package vault provides a HashiCorp Vault KV v2 secret provider for the Fluxor secrets API.
//
// Example:
//
//	cfg := vault.DefaultConfig()
//	cfg.Address = "https://vault.example.com:8200"
//	cfg.Token = os.Getenv("VAULT_TOKEN")
//	cfg.MountPath = "secret"
//	provider, err := vault.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//	value, err := provider.GetSecret("myapp/db")
package vault
