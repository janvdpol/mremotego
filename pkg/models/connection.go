package models

// Protocol represents the type of connection protocol
type Protocol string

const (
	ProtocolSSH     Protocol = "ssh"
	ProtocolRDP     Protocol = "rdp"
	ProtocolVNC     Protocol = "vnc"
	ProtocolHTTP    Protocol = "http"
	ProtocolHTTPS   Protocol = "https"
	ProtocolTelnet  Protocol = "telnet"
	ProtocolUnknown Protocol = "unknown"
)

// NodeType represents whether this is a connection or a folder
type NodeType string

const (
	NodeTypeConnection NodeType = "connection"
	NodeTypeFolder     NodeType = "folder"
)

// Connection represents a single connection or folder in the tree
type Connection struct {
	Name        string        `yaml:"name"`
	Type        NodeType      `yaml:"type"`
	Protocol    Protocol      `yaml:"protocol,omitempty"`
	Host        string        `yaml:"host,omitempty"`
	Port        int           `yaml:"port,omitempty"`
	Username    string        `yaml:"username,omitempty"`
	Password    string        `yaml:"password,omitempty"` // Consider encryption
	Domain      string        `yaml:"domain,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Children    []*Connection `yaml:"children,omitempty"`

	// Advanced options
	UseCredSSP bool   `yaml:"use_credssp,omitempty"`
	ColorDepth int    `yaml:"color_depth,omitempty"` // For RDP
	Resolution string `yaml:"resolution,omitempty"`  // For RDP
	ExtraArgs  string `yaml:"extra_args,omitempty"`  // Additional protocol-specific args

	// RDP Gateway options (RDP only)
	UseGateway         bool   `yaml:"use_gateway,omitempty"`
	GatewayUsageMethod string `yaml:"gateway_usage_method,omitempty"` // "never", "always", "detect"
	GatewayHostname    string `yaml:"gateway_hostname,omitempty"`
	GatewayCredentials string `yaml:"gateway_credentials,omitempty"` // "same", "different", "smartcard"
	GatewayUsername    string `yaml:"gateway_username,omitempty"`
	GatewayPassword    string `yaml:"gateway_password,omitempty"`
	GatewayDomain      string `yaml:"gateway_domain,omitempty"`

	// Metadata
	Tags     []string `yaml:"tags,omitempty"`
	Notes    string   `yaml:"notes,omitempty"`
	Created  string   `yaml:"created,omitempty"`
	Modified string   `yaml:"modified,omitempty"`
}

// Config represents the root configuration
type Config struct {
	Version     string        `yaml:"version"`
	Connections []*Connection `yaml:"connections"`
}

// NewConfig creates a new empty configuration
func NewConfig() *Config {
	return &Config{
		Version:     "1.0",
		Connections: make([]*Connection, 0),
	}
}

// NewConnection creates a new connection with default values
func NewConnection(name string, protocol Protocol) *Connection {
	return &Connection{
		Name:     name,
		Type:     NodeTypeConnection,
		Protocol: protocol,
	}
}

// NewFolder creates a new folder
func NewFolder(name string) *Connection {
	return &Connection{
		Name:     name,
		Type:     NodeTypeFolder,
		Children: make([]*Connection, 0),
	}
}

// IsFolder returns true if this node is a folder
func (c *Connection) IsFolder() bool {
	return c.Type == NodeTypeFolder
}

// AddChild adds a child connection or folder to this folder
func (c *Connection) AddChild(child *Connection) {
	if c.IsFolder() {
		c.Children = append(c.Children, child)
	}
}

// GetDefaultPort returns the default port for a protocol
func (p Protocol) GetDefaultPort() int {
	switch p {
	case ProtocolSSH:
		return 22
	case ProtocolRDP:
		return 3389
	case ProtocolVNC:
		return 5900
	case ProtocolHTTP:
		return 80
	case ProtocolHTTPS:
		return 443
	case ProtocolTelnet:
		return 23
	default:
		return 0
	}
}

// DeepCopy creates a deep copy of a Connection
func (c *Connection) DeepCopy() *Connection {
	if c == nil {
		return nil
	}

	connCopy := &Connection{
		Name:               c.Name,
		Type:               c.Type,
		Protocol:           c.Protocol,
		Host:               c.Host,
		Port:               c.Port,
		Username:           c.Username,
		Password:           c.Password,
		Domain:             c.Domain,
		Description:        c.Description,
		UseCredSSP:         c.UseCredSSP,
		ColorDepth:         c.ColorDepth,
		Resolution:         c.Resolution,
		ExtraArgs:          c.ExtraArgs,
		UseGateway:         c.UseGateway,
		GatewayUsageMethod: c.GatewayUsageMethod,
		GatewayHostname:    c.GatewayHostname,
		GatewayCredentials: c.GatewayCredentials,
		GatewayUsername:    c.GatewayUsername,
		GatewayPassword:    c.GatewayPassword,
		GatewayDomain:      c.GatewayDomain,
		Notes:              c.Notes,
		Created:            c.Created,
		Modified:           c.Modified,
	}

	// Deep copy tags
	if len(c.Tags) > 0 {
		connCopy.Tags = make([]string, len(c.Tags))
		copy(connCopy.Tags, c.Tags)
	}

	// Deep copy children
	if len(c.Children) > 0 {
		connCopy.Children = make([]*Connection, len(c.Children))
		for i, child := range c.Children {
			connCopy.Children[i] = child.DeepCopy()
		}
	}

	return connCopy
}

// DeepCopy creates a deep copy of a Config
func (cfg *Config) DeepCopy() *Config {
	if cfg == nil {
		return nil
	}

	cfgCopy := &Config{
		Version: cfg.Version,
	}

	// Deep copy connections
	if len(cfg.Connections) > 0 {
		cfgCopy.Connections = make([]*Connection, len(cfg.Connections))
		for i, conn := range cfg.Connections {
			cfgCopy.Connections[i] = conn.DeepCopy()
		}
	}

	return cfgCopy
}
