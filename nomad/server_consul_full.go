//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package nomad

import (
	"fmt"
	"net"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/command/agent/consul"
	"github.com/hashicorp/nomad/helper"
)

func (s *Server) setupBootstrapHandler() error {
	// peersTimeout is used to indicate to the Consul Syncer that the
	// current Nomad Server has a stale peer set.  peersTimeout will time
	// out if the Consul Syncer bootstrapFn has not observed a Raft
	// leader in maxStaleLeadership.  If peersTimeout has been triggered,
	// the Consul Syncer will begin querying Consul for other Nomad
	// Servers.
	//
	// NOTE: time.Timer is used vs time.Time in order to handle clock
	// drift because time.Timer is implemented as a monotonic clock.
	var peersTimeout *time.Timer = time.NewTimer(0)

	// consulQueryCount is the number of times the bootstrapFn has been
	// called, regardless of success.
	var consulQueryCount uint64

	// leadershipTimedOut is a helper method that returns true if the
	// peersTimeout timer has expired.
	leadershipTimedOut := func() bool {
		select {
		case <-peersTimeout.C:
			return true
		default:
			return false
		}
	}

	// The bootstrapFn callback handler is used to periodically poll
	// Consul to look up the Nomad Servers in Consul.  In the event the
	// server has been brought up without a `retry-join` configuration
	// and this Server is partitioned from the rest of the cluster,
	// periodically poll Consul to reattach this Server to other servers
	// in the same region and automatically reform a quorum (assuming the
	// correct number of servers required for quorum are present).
	bootstrapFn := func() error {
		// If there is a raft leader, do nothing
		if leader, _ := s.raft.LeaderWithID(); leader != "" {
			peersTimeout.Reset(maxStaleLeadership)
			return nil
		}

		// (ab)use serf.go's behavior of setting BootstrapExpect to
		// zero if we have bootstrapped.  If we have bootstrapped
		bootstrapExpect := s.config.BootstrapExpect
		if bootstrapExpect == 0 {
			// This Nomad Server has been bootstrapped.  Rely on
			// the peersTimeout firing as a guard to prevent
			// aggressive querying of Consul.
			if !leadershipTimedOut() {
				return nil
			}
		} else {
			if consulQueryCount > 0 && !leadershipTimedOut() {
				return nil
			}

			// This Nomad Server has not been bootstrapped, reach
			// out to Consul if our peer list is less than
			// `bootstrap_expect`.
			raftPeers, err := s.numPeers()
			if err != nil {
				peersTimeout.Reset(peersPollInterval + helper.RandomStagger(peersPollInterval/peersPollJitterFactor))
				return nil
			}

			// The necessary number of Nomad Servers required for
			// quorum has been reached, we do not need to poll
			// Consul.  Let the normal timeout-based strategy
			// take over.
			if raftPeers >= bootstrapExpect {
				peersTimeout.Reset(peersPollInterval + helper.RandomStagger(peersPollInterval/peersPollJitterFactor))
				return nil
			}
		}
		consulQueryCount++

		s.logger.Debug("lost contact with Nomad quorum, falling back to Consul for server list")

		dcs, err := s.consulCatalog.Datacenters()
		if err != nil {
			peersTimeout.Reset(peersPollInterval + helper.RandomStagger(peersPollInterval/peersPollJitterFactor))
			return fmt.Errorf("server.nomad: unable to query Consul datacenters: %v", err)
		}
		if len(dcs) > 2 {
			// Query the local DC first, then shuffle the
			// remaining DCs.  If additional calls to bootstrapFn
			// are necessary, this Nomad Server will eventually
			// walk all datacenter until it finds enough hosts to
			// form a quorum.
			shuffleStrings(dcs[1:])
			dcs = dcs[0:min(len(dcs), datacenterQueryLimit)]
		}

		nomadServerServiceName := s.config.GetDefaultConsul().ServerServiceName
		var mErr multierror.Error
		const defaultMaxNumNomadServers = 8
		nomadServerServices := make([]string, 0, defaultMaxNumNomadServers)
		localNode := s.serf.Memberlist().LocalNode()
		for _, dc := range dcs {
			consulOpts := &consulapi.QueryOptions{
				AllowStale: true,
				Datacenter: dc,
				Near:       "_agent",
				WaitTime:   consul.DefaultQueryWaitDuration,
			}
			consulServices, _, err := s.consulCatalog.Service(nomadServerServiceName, consul.ServiceTagSerf, consulOpts)
			if err != nil {
				err := fmt.Errorf("failed to query service %q in Consul datacenter %q: %v", nomadServerServiceName, dc, err)
				s.logger.Warn("failed to query Nomad service in Consul datacenter", "service_name", nomadServerServiceName, "dc", dc, "error", err)
				mErr.Errors = append(mErr.Errors, err)
				continue
			}

			for _, cs := range consulServices {
				port := strconv.FormatInt(int64(cs.ServicePort), 10)
				addr := cs.ServiceAddress
				if addr == "" {
					addr = cs.Address
				}
				if localNode.Addr.String() == addr && int(localNode.Port) == cs.ServicePort {
					continue
				}
				serverAddr := net.JoinHostPort(addr, port)
				nomadServerServices = append(nomadServerServices, serverAddr)
			}
		}

		if len(nomadServerServices) == 0 {
			if len(mErr.Errors) > 0 {
				peersTimeout.Reset(peersPollInterval + helper.RandomStagger(peersPollInterval/peersPollJitterFactor))
				return mErr.ErrorOrNil()
			}

			// Log the error and return nil so future handlers
			// can attempt to register the `nomad` service.
			pollInterval := peersPollInterval + helper.RandomStagger(peersPollInterval/peersPollJitterFactor)
			s.logger.Trace("no Nomad Servers advertising Nomad service in Consul datacenters", "service_name", nomadServerServiceName, "datacenters", dcs, "retry", pollInterval)
			peersTimeout.Reset(pollInterval)
			return nil
		}

		numServersContacted, err := s.Join(nomadServerServices)
		if err != nil {
			peersTimeout.Reset(peersPollInterval + helper.RandomStagger(peersPollInterval/peersPollJitterFactor))
			return fmt.Errorf("contacted %d Nomad Servers: %v", numServersContacted, err)
		}

		peersTimeout.Reset(maxStaleLeadership)
		s.logger.Info("successfully contacted Nomad servers", "num_servers", numServersContacted)

		return nil
	}

	// Hacky replacement for old ConsulSyncer Periodic Handler.
	go func() {
		lastOk := true
		sync := time.NewTimer(0)
		for {
			select {
			case <-sync.C:
				d := defaultConsulDiscoveryInterval
				if err := bootstrapFn(); err != nil {
					// Only log if it worked last time
					if lastOk {
						lastOk = false
						s.logger.Error("error looking up Nomad servers in Consul", "error", err)
					}
					d = defaultConsulDiscoveryIntervalRetry
				}
				sync.Reset(d)
			case <-s.shutdownCh:
				return
			}
		}
	}()
	return nil
}

// setupConsulSyncer creates Server-mode consul.Syncer which periodically
// executes callbacks on a fixed interval.
func (s *Server) setupConsulSyncer() error {
	conf := s.config.GetDefaultConsul()
	if conf.ServerAutoJoin != nil && *conf.ServerAutoJoin {
		if err := s.setupBootstrapHandler(); err != nil {
			return err
		}
	}

	return nil
}
