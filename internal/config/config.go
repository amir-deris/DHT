package config

import (
	"errors"
	"fmt"
	"strings"
)

// Config captures node runtime configuration.
type Config struct {
	NodeID            string
	BindAddr          string
	SeedsCSV          string
	Seeds             []string
	ReplicationFactor int
	ReadQuorum        int
	WriteQuorum       int
}

// Flags returns a zero-value config for flag binding.
func Flags() *Config {
	return &Config{}
}

// Validate finalizes and validates the configuration.
func (c *Config) Validate() error {
	if c.NodeID == "" {
		// Default node id to hostname if available
		c.NodeID = generateDefaultNodeID()
	}
	if c.BindAddr == "" {
		c.BindAddr = ":8080"
	}
	if c.ReplicationFactor <= 0 {
		c.ReplicationFactor = 3
	}
	if c.ReadQuorum <= 0 {
		c.ReadQuorum = 2
	}
	if c.WriteQuorum <= 0 {
		c.WriteQuorum = 2
	}
	if c.ReadQuorum > c.ReplicationFactor || c.WriteQuorum > c.ReplicationFactor {
		return fmt.Errorf("R and W must be <= N (R=%d W=%d N=%d)", c.ReadQuorum, c.WriteQuorum, c.ReplicationFactor)
	}
	if c.SeedsCSV != "" {
		parts := strings.Split(c.SeedsCSV, ",")
		for _, p := range parts {
			s := strings.TrimSpace(p)
			if s != "" {
				c.Seeds = append(c.Seeds, s)
			}
		}
	}
	if c.NodeID == "" {
		return errors.New("node-id must be set or resolvable from hostname")
	}
	return nil
}

func generateDefaultNodeID() string {
	// For now, hostname is sufficient; later we may compose with a short ID
	if h, err := osHostname(); err == nil && h != "" {
		return h
	}
	return "node-unknown"
}

// osHostname exists for testability.
var osHostname = func() (string, error) { return getHostname() }

func getHostname() (string, error) { return hostname() }

// indirection to avoid importing os in tests directly here if needed
func hostname() (string, error) { return osHostnameImpl() }

var osHostnameImpl = func() (string, error) { return osHostnameReal() }

// real implementation in a separate file to keep this file minimal.
