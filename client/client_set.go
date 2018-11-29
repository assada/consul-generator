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

type ClientSet struct {
	sync.RWMutex

	consul *consulClient
}

type consulClient struct {
	client    *consulapi.Client
	transport *http.Transport
}

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

func NewClientSet() *ClientSet {
	return &ClientSet{}
}

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

	if i.SSLEnabled {
		consulConfig.Scheme = "https"

		var tlsConfig tls.Config

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

		if i.SSLCACert != "" || i.SSLCAPath != "" {
			rootConfig := &rootcerts.Config{
				CAFile: i.SSLCACert,
				CAPath: i.SSLCAPath,
			}
			if err := rootcerts.ConfigureTLS(&tlsConfig, rootConfig); err != nil {
				return fmt.Errorf("client set: consul configuring TLS failed: %s", err)
			}
		}

		tlsConfig.BuildNameToCertificate()

		if i.ServerName != "" {
			tlsConfig.ServerName = i.ServerName
			tlsConfig.InsecureSkipVerify = false
		}
		if !i.SSLVerify {
			log.Printf("[WARN] (clients) disabling consul SSL verification")
			tlsConfig.InsecureSkipVerify = true
		}

		transport.TLSClientConfig = &tlsConfig
	}

	consulConfig.Transport = transport

	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return fmt.Errorf("client set: consul: %s", err)
	}

	c.Lock()
	c.consul = &consulClient{
		client:    client,
		transport: transport,
	}
	c.Unlock()

	return nil
}

func (c *ClientSet) Consul() *consulapi.Client {
	c.RLock()
	defer c.RUnlock()
	return c.consul.client
}

func (c *ClientSet) Stop() {
	c.Lock()
	defer c.Unlock()

	if c.consul != nil {
		c.consul.transport.CloseIdleConnections()
	}
}
