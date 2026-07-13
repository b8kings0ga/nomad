//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-secure-stdlib/listenerutil"
	"github.com/hashicorp/nomad/helper/pointer"
	"github.com/hashicorp/nomad/nomad/structs"
)

func DefaultConsulConfig() *ConsulConfig {
	def := consul.DefaultConfig()
	return defaultConsulConfig(def.Address, def.Scheme == "https", !def.TLSConfig.InsecureSkipVerify, def.TLSConfig.CAFile, def.Namespace, def.Token)
}

func defaultConsulConfig(addr string, ssl, verify bool, caFile, namespace, token string) *ConsulConfig {
	return &ConsulConfig{
		Name: "default", ServerServiceName: "nomad", ServerHTTPCheckName: "Nomad Server HTTP Check",
		ServerSerfCheckName: "Nomad Server Serf Check", ServerRPCCheckName: "Nomad Server RPC Check",
		ClientServiceName: "nomad-client", ClientHTTPCheckName: "Nomad Client HTTP Check",
		AutoAdvertise: pointer.Of(true), ChecksUseAdvertise: pointer.Of(false),
		ServerAutoJoin: pointer.Of(true), ClientAutoJoin: pointer.Of(true), Timeout: 5 * time.Second,
		ServiceIdentityAuthMethod: structs.ConsulWorkloadsDefaultAuthMethodName,
		TaskIdentityAuthMethod:    structs.ConsulWorkloadsDefaultAuthMethodName,
		Addr:                      addr, EnableSSL: pointer.Of(ssl), VerifySSL: pointer.Of(verify), CAFile: caFile, Namespace: namespace, Token: token,
	}
}

func (c *ConsulConfig) ApiConfig() (*consul.Config, error) {
	config := consul.DefaultConfig()
	if c.Addr != "" {
		ipStr, err := listenerutil.ParseSingleIPTemplate(c.Addr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse address template %q: %v", c.Addr, err)
		}
		config.Address = ipStr
	}
	if c.Token != "" {
		config.Token = c.Token
	}
	if c.Timeout != 0 {
		if config.HttpClient == nil {
			config.HttpClient = &http.Client{}
		}
		config.HttpClient.Timeout = c.Timeout
		config.HttpClient.Transport = config.Transport
	}
	if c.Auth != "" {
		username, password := c.Auth, ""
		if strings.Contains(c.Auth, ":") {
			parts := strings.SplitN(c.Auth, ":", 2)
			username, password = parts[0], parts[1]
		}
		config.HttpAuth = &consul.HttpBasicAuth{Username: username, Password: password}
	}
	if c.EnableSSL != nil && *c.EnableSSL {
		config.Scheme = "https"
		config.TLSConfig = consul.TLSConfig{Address: config.Address, CAFile: c.CAFile, CertFile: c.CertFile, KeyFile: c.KeyFile}
		if c.VerifySSL != nil {
			config.TLSConfig.InsecureSkipVerify = !*c.VerifySSL
		}
		tlsConfig, err := consul.SetupTLSConfig(&config.TLSConfig)
		if err != nil {
			return nil, err
		}
		config.Transport.TLSClientConfig = tlsConfig
	}
	if c.Namespace != "" {
		config.Namespace = c.Namespace
	}
	return config, nil
}
