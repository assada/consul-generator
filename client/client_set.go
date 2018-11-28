package client

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	rootcerts "github.com/hashicorp/go-rootcerts"
)

// ClientSet is a collection of clients that dependencies use to communicate
// with remote services like Consul or Vault.
type ClientSet struct {
	sync.RWMutex

	consul *consulClient
}

// consulClient is a wrapper around a real Consul API client.
type consulClient struct {
	client    *consulapi.Client
	transport *http.Transport
}

// CreateConsulClientInput is used as input to the CreateConsulClient function.
type CreateConsulClientInput struct {
	Address      string
	Token        string
	AuthEnabled  bool
	AuthUsername string
	AuthPassword string
	SSLEnabled   bool
	SSLVerify    bool
	SSLCert      string
	SSLKey       string
	SSLCACert    string
	SSLCAPath    string
	ServerName   string

	TransportDialKeepAlive       time.Duration
	TransportDialTimeout         time.Duration
	TransportDisableKeepAlives   bool
	TransportIdleConnTimeout     time.Duration
	TransportMaxIdleConns        int
	TransportMaxIdleConnsPerHost int
	TransportTLSHandshakeTimeout time.Duration
}

// CreateVaultClientInput is used as input to the CreateVaultClient function.
type CreateVaultClientInput struct {
	Address     string
	Token       string
	UnwrapToken bool
	SSLEnabled  bool
	SSLVerify   bool
	SSLCert     string
	SSLKey      string
	SSLCACert   string
	SSLCAPath   string
	ServerName  string

	TransportDialKeepAlive       time.Duration
	TransportDialTimeout         time.Duration
	TransportDisableKeepAlives   bool
	TransportIdleConnTimeout     time.Duration
	TransportMaxIdleConns        int
	TransportMaxIdleConnsPerHost int
	TransportTLSHandshakeTimeout time.Duration
}

// NewClientSet creates a new client set that is ready to accept clients.
func NewClientSet() *ClientSet {
	return &ClientSet{}
}

// CreateConsulClient creates a new Consul API client from the given input.
func (c *ClientSet) CreateConsulClient(i *CreateConsulClientInput) error {
	consulConfig := consulapi.DefaultConfig()

	if i.Address != "" {
		consulConfig.Address = i.Address
	}

	if i.Token != "" {
		consulConfig.Token = i.Token
	}

	if i.AuthEnabled {
		consulConfig.HttpAuth = &consulapi.HttpBasicAuth{
			Username: i.AuthUsername,
			Password: i.AuthPassword,
		}
	}

	// This transport will attempt to keep connections open to the Consul server.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   i.TransportDialTimeout,
			KeepAlive: i.TransportDialKeepAlive,
		}).Dial,
		DisableKeepAlives:   i.TransportDisableKeepAlives,
		MaxIdleConns:        i.TransportMaxIdleConns,
		IdleConnTimeout:     i.TransportIdleConnTimeout,
		MaxIdleConnsPerHost: i.TransportMaxIdleConnsPerHost,
		TLSHandshakeTimeout: i.TransportTLSHandshakeTimeout,
	}

	// Configure SSL
	if i.SSLEnabled {
		consulConfig.Scheme = "https"

		var tlsConfig tls.Config

		// Custom certificate or certificate and key
		if i.SSLCert != "" && i.SSLKey != "" {
			cert, err := tls.LoadX509KeyPair(i.SSLCert, i.SSLKey)
			if err != nil {
				return fmt.Errorf("client set: consul: %s", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		} else if i.SSLCert != "" {
			cert, err := tls.LoadX509KeyPair(i.SSLCert, i.SSLCert)
			if err != nil {
				return fmt.Errorf("client set: consul: %s", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Custom CA certificate
		if i.SSLCACert != "" || i.SSLCAPath != "" {
			rootConfig := &rootcerts.Config{
				CAFile: i.SSLCACert,
				CAPath: i.SSLCAPath,
			}
			if err := rootcerts.ConfigureTLS(&tlsConfig, rootConfig); err != nil {
				return fmt.Errorf("client set: consul configuring TLS failed: %s", err)
			}
		}

		// Construct all the certificates now
		tlsConfig.BuildNameToCertificate()

		// SSL verification
		if i.ServerName != "" {
			tlsConfig.ServerName = i.ServerName
			tlsConfig.InsecureSkipVerify = false
		}
		if !i.SSLVerify {
			log.Printf("[WARN] (clients) disabling consul SSL verification")
			tlsConfig.InsecureSkipVerify = true
		}

		// Save the TLS config on our transport
		transport.TLSClientConfig = &tlsConfig
	}

	// Setup the new transport
	consulConfig.Transport = transport

	// Create the API client
	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return fmt.Errorf("client set: consul: %s", err)
	}

	// Save the data on ourselves
	c.Lock()
	c.consul = &consulClient{
		client:    client,
		transport: transport,
	}
	c.Unlock()

	return nil
}

// Consul returns the Consul client for this set.
func (c *ClientSet) Consul() *consulapi.Client {
	c.RLock()
	defer c.RUnlock()
	return c.consul.client
}

// Stop closes all idle connections for any attached clients.
func (c *ClientSet) Stop() {
	c.Lock()
	defer c.Unlock()

	if c.consul != nil {
		c.consul.transport.CloseIdleConnections()
	}
}
