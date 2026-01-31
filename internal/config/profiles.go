// internal/config/profiles.go
package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GetProfile retrieves a profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", name)
}

// AddProfile adds a new profile to the config
func (c *Config) AddProfile(p Profile) error {
	for _, existing := range c.Profiles {
		if existing.Name == p.Name {
			return fmt.Errorf("profile already exists: %s", p.Name)
		}
	}
	c.Profiles = append(c.Profiles, p)
	return c.Save()
}

// UpdateProfile updates an existing profile
func (c *Config) UpdateProfile(name string, p Profile) error {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			c.Profiles[i] = p
			return c.Save()
		}
	}
	return fmt.Errorf("profile not found: %s", name)
}

// DeleteProfile removes a profile from the config
func (c *Config) DeleteProfile(name string) error {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			return c.Save()
		}
	}
	return fmt.Errorf("profile not found: %s", name)
}

// ListProfiles returns all profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, len(c.Profiles))
	for i, p := range c.Profiles {
		names[i] = p.Name
	}
	return names
}

// BuildDSN builds a database connection string from profile and password
// Returns URI format for display (mysql://..., postgres://..., sqlite://...)
func (p *Profile) BuildDSN(password string) string {
	// Use provided password, or fall back to profile's stored password
	if password == "" {
		password = p.Password
	}
	switch p.Type {
	case "postgres":
		if password != "" {
			return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", p.User, password, p.Host, p.Port, p.Database)
		}
		return fmt.Sprintf("postgres://%s@%s:%d/%s", p.User, p.Host, p.Port, p.Database)
	case "mysql":
		if password != "" {
			return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", p.User, password, p.Host, p.Port, p.Database)
		}
		return fmt.Sprintf("mysql://%s@%s:%d/%s", p.User, p.Host, p.Port, p.Database)
	case "sqlite":
		return fmt.Sprintf("sqlite://%s", p.Database)
	default:
		return ""
	}
}

// BuildDriverDSN builds a driver-specific connection string for actual database connection
func (p *Profile) BuildDriverDSN(password string) string {
	if password == "" {
		password = p.Password
	}
	switch p.Type {
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", p.User, password, p.Host, p.Port, p.Database)
	case "mysql":
		// MySQL driver uses different format: user:pass@tcp(host:port)/db
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", p.User, password, p.Host, p.Port, p.Database)
	case "sqlite":
		return fmt.Sprintf("file:%s", p.Database)
	default:
		return ""
	}
}

// ParseDSN parses a connection string into a Profile
func ParseDSN(name, dsn string) (Profile, error) {
	p := Profile{Name: name}

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return p, err
		}
		p.Type = "postgres"
		p.Host = u.Hostname()
		port := u.Port()
		if port == "" {
			p.Port = 5432
		} else {
			p.Port, _ = strconv.Atoi(port)
		}
		p.User = u.User.Username()
		p.Password, _ = u.User.Password()
		p.Database = strings.TrimPrefix(u.Path, "/")
	} else if strings.HasPrefix(dsn, "mysql://") {
		// mysql://user:pass@host:port/db
		u, err := url.Parse(dsn)
		if err != nil {
			return p, err
		}
		p.Type = "mysql"
		p.Host = u.Hostname()
		port := u.Port()
		if port == "" {
			p.Port = 3306
		} else {
			p.Port, _ = strconv.Atoi(port)
		}
		p.User = u.User.Username()
		p.Password, _ = u.User.Password()
		p.Database = strings.TrimPrefix(u.Path, "/")
	} else if strings.HasPrefix(dsn, "sqlite://") || strings.HasPrefix(dsn, "file:") {
		// sqlite:///path/to.db or file:test.db
		p.Type = "sqlite"
		path := strings.TrimPrefix(dsn, "sqlite://")
		path = strings.TrimPrefix(path, "file:")
		p.Database = path // For SQLite, Database field holds the path
	} else {
		// Assume SQLite file path if no scheme match
		p.Type = "sqlite"
		p.Database = dsn
	}

	return p, nil
}
