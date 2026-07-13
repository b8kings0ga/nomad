//go:build linux && !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package allocrunner

import (
	"fmt"
	"slices"
	"strconv"

	consulIPTables "github.com/hashicorp/consul/sdk/iptables"
	"github.com/hashicorp/go-set/v3"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/helper/envoy"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/plugins/drivers"
)

// setupTransparentProxyArgs returns a Consul SDK iptables configuration if the
// allocation has a transparent_proxy block.
func (c *cniNetworkConfigurator) setupTransparentProxyArgs(
	alloc *structs.Allocation,
	spec *drivers.NetworkIsolationSpec,
	portMaps *portMappings,
) (*transparentProxyArgs, error) {
	var tproxy *structs.ConsulTransparentProxy
	var cluster, proxyUID string
	var proxyInboundPort, proxyOutboundPort int
	var exposePorts []string
	outboundPorts := []string{}

	tg := alloc.Job.LookupTaskGroup(alloc.TaskGroup)
	for _, svc := range tg.Services {
		if !svc.Connect.HasTransparentProxy() {
			continue
		}

		tproxy = svc.Connect.SidecarService.Proxy.TransparentProxy
		cluster = svc.Cluster
		proxyUID = c.nodeMeta[envoy.DefaultTransparentProxyUIDParam]
		if tproxy.UID != "" {
			proxyUID = tproxy.UID
		}

		if tproxy.OutboundPort != 0 {
			proxyOutboundPort = int(tproxy.OutboundPort)
		} else {
			outboundPortAttr := c.nodeMeta[envoy.DefaultTransparentProxyOutboundPortParam]
			parsedOutboundPort, err := strconv.ParseUint(outboundPortAttr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf(
					"could not parse default_outbound_port %q as port number: %w",
					outboundPortAttr, err)
			}
			proxyOutboundPort = int(parsedOutboundPort)
		}

		envoyPortLabel := "connect-proxy-" + svc.Name
		if envoyPort, ok := portMaps.get(envoyPortLabel); ok {
			proxyInboundPort = int(envoyPort.HostPort)
		}

		if len(tproxy.ExcludeOutboundPorts) == 0 {
			outboundPorts = nil
		} else {
			outboundPorts = helper.ConvertSlice(tproxy.ExcludeOutboundPorts,
				func(p uint16) string { return fmt.Sprint(p) })
		}

		exposePortSet := set.From(exposePorts)
		for _, network := range tg.Networks {
			for _, port := range network.ReservedPorts {
				exposePortSet.Insert(fmt.Sprint(port.To))
			}
		}

		for _, portLabel := range tproxy.ExcludeInboundPorts {
			if _, err := strconv.ParseUint(portLabel, 10, 16); err == nil {
				exposePortSet.Insert(portLabel)
				continue
			}
			if port, ok := portMaps.get(portLabel); ok {
				exposePortSet.Insert(strconv.FormatInt(int64(port.ContainerPort), 10))
			}
		}

		if svc.Connect.SidecarService.Proxy.Expose != nil {
			for _, path := range svc.Connect.SidecarService.Proxy.Expose.Paths {
				if port, ok := portMaps.get(path.ListenerPort); ok {
					exposePortSet.Insert(strconv.FormatInt(int64(port.ContainerPort), 10))
				}
			}
		}

		if exposePortSet.Size() > 0 {
			exposePorts = exposePortSet.Slice()
			slices.Sort(exposePorts)
		}
		break
	}

	if tproxy == nil {
		return nil, nil
	}

	var dnsAddr string
	var dnsPort int
	if !tproxy.NoDNS {
		dnsAddr, dnsPort = c.dnsFromAttrs(cluster)
	}

	config := &consulIPTables.Config{
		ConsulDNSIP:          dnsAddr,
		ConsulDNSPort:        dnsPort,
		ProxyUserID:          proxyUID,
		ProxyInboundPort:     proxyInboundPort,
		ProxyOutboundPort:    proxyOutboundPort,
		ExcludeInboundPorts:  exposePorts,
		ExcludeOutboundPorts: outboundPorts,
		ExcludeOutboundCIDRs: tproxy.ExcludeOutboundCIDRs,
		ExcludeUIDs:          tproxy.ExcludeUIDs,
		NetNS:                spec.Path,
	}
	return &transparentProxyArgs{config: config, consulDNSIP: dnsAddr}, nil
}

func (c *cniNetworkConfigurator) dnsFromAttrs(cluster string) (string, int) {
	var dnsAddrAttr, dnsPortAttr string
	if cluster == structs.ConsulDefaultCluster || cluster == "" {
		dnsAddrAttr = "unique.consul.dns.addr"
		dnsPortAttr = "consul.dns.port"
	} else {
		dnsAddrAttr = "unique.consul." + cluster + ".dns.addr"
		dnsPortAttr = "consul." + cluster + ".dns.port"
	}

	dnsAddr, ok := c.nodeAttrs[dnsAddrAttr]
	if !ok || dnsAddr == "" {
		return "", 0
	}
	dnsPort, ok := c.nodeAttrs[dnsPortAttr]
	if !ok || dnsPort == "0" || dnsPort == "-1" {
		return "", 0
	}
	port, err := strconv.ParseUint(dnsPort, 10, 16)
	if err != nil {
		return "", 0
	}
	return dnsAddr, int(port)
}
