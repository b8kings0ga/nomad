//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"fmt"
	"net"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/client/servers"
	"github.com/hashicorp/nomad/command/agent/consul"
)

func (c *Client) consulDiscoveryImpl() error {
	consulLogger := c.logger.Named("consul")

	dcs, err := c.consulCatalog.Datacenters()
	if err != nil {
		return fmt.Errorf("client.consul: unable to query Consul datacenters: %v", err)
	}
	if len(dcs) > 2 {
		// Query the local DC first, then shuffle the
		// remaining DCs.  Future heartbeats will cause Nomad
		// Clients to fixate on their local datacenter so
		// it's okay to talk with remote DCs.  If the no
		// Nomad servers are available within
		// datacenterQueryLimit, the next heartbeat will pick
		// a new set of servers so it's okay.
		shuffleStrings(dcs[1:])
		dcs = dcs[0:min(len(dcs), datacenterQueryLimit)]
	}

	serviceName := c.GetConfig().GetDefaultConsul().ServerServiceName
	var mErr multierror.Error
	var nomadServers servers.Servers
	consulLogger.Debug("bootstrap contacting Consul DCs", "consul_dcs", dcs)
DISCOLOOP:
	for _, dc := range dcs {
		consulOpts := &consulapi.QueryOptions{
			AllowStale: true,
			Datacenter: dc,
			Near:       "_agent",
			WaitTime:   consul.DefaultQueryWaitDuration,
		}
		consulServices, _, err := c.consulCatalog.Service(serviceName, consul.ServiceTagRPC, consulOpts)
		if err != nil {
			mErr.Errors = append(mErr.Errors, fmt.Errorf("unable to query service %+q from Consul datacenter %+q: %v", serviceName, dc, err))
			continue
		}

		for _, s := range consulServices {
			port := strconv.Itoa(s.ServicePort)
			addrstr := s.ServiceAddress
			if addrstr == "" {
				addrstr = s.Address
			}
			addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(addrstr, port))
			if err != nil {
				mErr.Errors = append(mErr.Errors, err)
				continue
			}

			srv := &servers.Server{Addr: addr}
			nomadServers = append(nomadServers, srv)
		}

		if len(nomadServers) > 0 {
			break DISCOLOOP
		}

	}
	if len(nomadServers) == 0 {
		if len(mErr.Errors) > 0 {
			return mErr.ErrorOrNil()
		}
		return fmt.Errorf("no Nomad Servers advertising service %q in Consul datacenters: %+q", serviceName, dcs)
	}

	consulLogger.Info("discovered following servers", "servers", nomadServers)

	// Fire the retry trigger if we have updated the set of servers.
	if c.servers.SetServers(nomadServers) {
		// Start rebalancing
		c.servers.RebalanceServers()

		// Notify waiting rpc calls. If a goroutine just failed an RPC call and
		// isn't receiving on this chan yet they'll still retry eventually.
		// This is a shortcircuit for the longer retry intervals.
		c.fireRpcRetryWatcher()
	}

	return nil
}

// setupStatsLabels builds the base labels for the client. This should be done
// before Client.run() is called, so that sub-system can use these without fear
// they have not been set.
