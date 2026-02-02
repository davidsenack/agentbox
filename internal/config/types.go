package config

// Config represents the agentbox.yaml configuration
type Config struct {
	Runtime string        `yaml:"runtime"`
	VM      VMConfig      `yaml:"vm"`
	Network NetworkConfig `yaml:"network"`
	Secrets SecretsConfig `yaml:"secrets"`
	Mounts  []MountConfig `yaml:"mounts"`
}

// VMConfig defines virtual machine settings
type VMConfig struct {
	CPUs   int    `yaml:"cpus"`
	Memory string `yaml:"memory"`
	Disk   string `yaml:"disk"`
}

// NetworkConfig defines network settings
type NetworkConfig struct {
	ProxyPort  int          `yaml:"proxy_port"`
	InjectAuth []AuthConfig `yaml:"inject_auth"`
}

// AuthConfig defines proxy-injected authentication
// The secret is read from host env and injected by proxy - never enters VM
type AuthConfig struct {
	Host   string `yaml:"host"`   // Target host (e.g., api.anthropic.com)
	Header string `yaml:"header"` // Header name (e.g., x-api-key)
	Env    string `yaml:"env"`    // Host env var to read (e.g., ANTHROPIC_API_KEY)
}

// SecretsConfig defines secret handling settings
type SecretsConfig struct {
	RedactPatterns []string `yaml:"redact_patterns"`
}

// MountConfig defines a host-to-guest mount
type MountConfig struct {
	Host     string `yaml:"host"`
	Guest    string `yaml:"guest"`
	Writable bool   `yaml:"writable"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Runtime: "lima",
		VM: VMConfig{
			CPUs:   4,
			Memory: "4GiB",
			Disk:   "30GiB",
		},
		Network: NetworkConfig{
			ProxyPort: 3128,
			InjectAuth: []AuthConfig{
				{
					Host:   "api.anthropic.com",
					Header: "x-api-key",
					Env:    "ANTHROPIC_API_KEY",
				},
			},
		},
		Secrets: SecretsConfig{
			RedactPatterns: []string{
				`sk-ant-[a-zA-Z0-9-]+`,
				`sk-[a-zA-Z0-9]{48}`,
			},
		},
		Mounts: []MountConfig{
			{Host: "./workspace", Guest: "/workspace", Writable: true},
			{Host: "./artifacts", Guest: "/artifacts", Writable: true},
		},
	}
}
