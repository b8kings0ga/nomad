// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

// For now CNI is supported only on Linux.
//
//go:build linux
// +build linux

package allocrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	cni "github.com/containerd/go-cni"
	cnilibrary "github.com/containernetworking/cni/libcni"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/nomad/client/taskenv"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/plugins/drivers"
)

const (

	// envCNIPath is the environment variable name to use to derive the CNI path
	// when it is not explicitly set by the client
	envCNIPath = "CNI_PATH"

	// defaultCNIPath is the CNI path to use when it is not set by the client
	// and is not set by environment variable
	defaultCNIPath = "/opt/cni/bin"

	// defaultCNIInterfacePrefix is the network interface to use if not set in
	// client config
	defaultCNIInterfacePrefix = "eth"
)

type cniNetworkConfigurator struct {
	cni                     cni.CNI
	confParser              *cniConfParser
	ignorePortMappingHostIP bool
	nodeAttrs               map[string]string
	nodeMeta                map[string]string
	rand                    *rand.Rand
	logger                  log.Logger
	nsOpts                  *nsOpts
	newIPTables             func(structs.NodeNetworkAF) (IPTablesCleanup, error)
}

func newCNINetworkConfigurator(logger log.Logger, cniPath, cniInterfacePrefix, cniConfDir, networkName string, ignorePortMappingHostIP bool, node *structs.Node) (*cniNetworkConfigurator, error) {
	parser, err := loadCNIConf(cniConfDir, networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to load CNI config: %v", err)
	}

	return newCNINetworkConfiguratorWithConf(logger, cniPath, cniInterfacePrefix, ignorePortMappingHostIP, parser, node)
}

func newCNINetworkConfiguratorWithConf(logger log.Logger, cniPath, cniInterfacePrefix string, ignorePortMappingHostIP bool, parser *cniConfParser, node *structs.Node) (*cniNetworkConfigurator, error) {
	conf := &cniNetworkConfigurator{
		confParser:              parser,
		rand:                    rand.New(rand.NewSource(time.Now().Unix())),
		logger:                  logger,
		ignorePortMappingHostIP: ignorePortMappingHostIP,
		nodeAttrs:               node.Attributes,
		nodeMeta:                node.Meta,
		nsOpts:                  &nsOpts{},
		newIPTables:             newIPTablesCleanup,
	}
	if cniPath == "" {
		if cniPath = os.Getenv(envCNIPath); cniPath == "" {
			cniPath = defaultCNIPath
		}
	}

	if cniInterfacePrefix == "" {
		cniInterfacePrefix = defaultCNIInterfacePrefix
	}

	c, err := cni.New(cni.WithPluginDir(filepath.SplitList(cniPath)),
		cni.WithInterfacePrefix(cniInterfacePrefix))
	if err != nil {
		return nil, err
	}
	conf.cni = c

	return conf, nil
}

const (
	ConsulIPTablesConfigEnvVar = "CONSUL_IPTABLES_CONFIG"
)

type transparentProxyArgs struct {
	config      any
	consulDNSIP string
}

// Adds user inputted custom CNI args to cniArgs map
func addCustomCNIArgs(networks []*structs.NetworkResource, cniArgs map[string]string) {
	for _, net := range networks {
		if net.CNI == nil {
			continue
		}
		for k, v := range net.CNI.Args {
			cniArgs[k] = v
		}
	}
}

func addNomadWorkloadCNIArgs(logger log.Logger, alloc *structs.Allocation, cniArgs map[string]string) {
	for key, value := range map[string]string{
		// these are the very same keys that are used to build task env vars
		taskenv.Region:    alloc.Job.Region, // NOMAD_REGION
		taskenv.Namespace: alloc.Namespace,  // NOMAD_NAMESPACE
		taskenv.JobID:     alloc.Job.ID,     // NOMAD_JOB_ID
		taskenv.GroupName: alloc.TaskGroup,  // NOMAD_GROUP_NAME
		taskenv.AllocID:   alloc.ID,         // NOMAD_ALLOC_ID
	} {
		// job ID and group name may contain ";" but CNI_ARGS are ";"-separated
		// per the spec, so they may not be used in arg keys or values.
		if strings.Contains(value, ";") {
			logger.Warn("Skipping CNI arg because it contains a semicolon",
				"key", key, "value", value)
		} else {
			cniArgs[key] = value
		}
	}
}

var supportsCNICheck = mustCNICheckConstraint()

func mustCNICheckConstraint() version.Constraints {
	v, err := version.NewConstraint(">= 1.3.0")
	if err != nil {
		panic(err)
	}
	return v
}

// Setup calls the CNI plugins with the add action
func (c *cniNetworkConfigurator) Setup(ctx context.Context, alloc *structs.Allocation, spec *drivers.NetworkIsolationSpec, created bool) (*structs.AllocNetworkStatus, error) {

	if err := c.ensureCNIInitialized(); err != nil {
		return nil, fmt.Errorf("cni not initialized: %w", err)
	}
	cniArgs := map[string]string{
		// CNI plugins are called one after the other with the same set of
		// arguments. Passing IgnoreUnknown=true signals to plugins that they
		// should ignore any arguments they don't understand
		"IgnoreUnknown": "true",
	}

	tg := alloc.Job.LookupTaskGroup(alloc.TaskGroup)

	addCustomCNIArgs(tg.Networks, cniArgs)

	// Add NOMAD_* after custom args so it cannot be overridden.
	addNomadWorkloadCNIArgs(c.logger, alloc, cniArgs)

	portMaps := getPortMapping(alloc, c.ignorePortMappingHostIP)

	tproxyArgs, err := c.setupTransparentProxyArgs(alloc, spec, portMaps)
	if err != nil {
		return nil, err
	}
	if tproxyArgs != nil {
		iptablesCfg, err := json.Marshal(tproxyArgs.config)
		if err != nil {
			return nil, err
		}
		cniArgs[ConsulIPTablesConfigEnvVar] = string(iptablesCfg)
	}

	if !created {
		// The netns will not be created if it already exists, typically on
		// agent restart. If the configuration of a prexisting netns is wrong
		// (ex. after a host reboot for docker created netns), networking will
		// be broken. CNI's ADD command is not idempotent so we can't simply try
		// again. Run CHECK to verify the network is still valid. Older plugins
		// have a broken CHECK, so we have to allow the buggy behavior in the
		// case of a host reboot with docker-created netns there.
		cniVersion, err := version.NewSemver(c.nodeAttrs["plugins.cni.version.bridge"])
		if err == nil && supportsCNICheck.Check(cniVersion) {
			err := c.cni.Check(ctx, alloc.ID, spec.Path,
				c.nsOpts.withCapabilityPortMap(portMaps.ports),
				c.nsOpts.withArgs(cniArgs),
			)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrCNICheckFailed, err)
			}
		} else {
			c.logger.Debug("network namespace exists but could not check if networking is valid because bridge plugin version was <1.3.0: continuing anyways")
			return nil, nil
		}
		c.logger.Trace("network namespace exists and passed check: skipping setup")
		return nil, nil
	}

	// Depending on the version of bridge cni plugin used, a known race could occure
	// where two alloc attempt to create the nomad bridge at the same time, resulting
	// in one of them to fail. This rety attempts to overcome those erroneous failures.
	const retry = 3
	var firstError error
	var res *cni.Result
	for attempt := 1; ; attempt++ {
		var err error
		if res, err = c.cni.Setup(ctx, alloc.ID, spec.Path,
			c.nsOpts.withCapabilityPortMap(portMaps.ports),
			c.nsOpts.withArgs(cniArgs),
		); err != nil {
			c.logger.Warn("failed to configure network", "error", err, "attempt", attempt)
			switch attempt {
			case 1:
				firstError = err
			case retry:
				return nil, fmt.Errorf("failed to configure network: %v", firstError)
			}

			// Sleep for 1 second + jitter
			time.Sleep(time.Second + (time.Duration(c.rand.Int63n(1000)) * time.Millisecond))
			continue
		}
		break
	}

	if c.logger.IsDebug() {
		resultJSON, _ := json.Marshal(res)
		c.logger.Debug("received result from CNI", "result", string(resultJSON))
	}

	allocNet, err := c.cniToAllocNet(res)
	if err != nil {
		return nil, err
	}

	// overwrite the nameservers with Consul DNS, if we have it; we don't need
	// the port because the iptables rule redirects port 53 traffic to it
	if tproxyArgs != nil && tproxyArgs.consulDNSIP != "" {
		if allocNet.DNS == nil {
			allocNet.DNS = &structs.DNSConfig{
				Servers:  []string{},
				Searches: []string{},
				Options:  []string{},
			}
		}
		allocNet.DNS.Servers = []string{tproxyArgs.consulDNSIP}
	}

	return allocNet, nil
}

// cniToAllocNet converts a cni.Result to an AllocNetworkStatus or returns an
// error. The first interface and IP with a sandbox and address set are
// preferred. Failing that the first interface with an IP is selected.
func (c *cniNetworkConfigurator) cniToAllocNet(res *cni.Result) (*structs.AllocNetworkStatus, error) {
	if len(res.Interfaces) == 0 {
		return nil, fmt.Errorf("failed to configure network: no interfaces found")
	}

	netStatus := new(structs.AllocNetworkStatus)

	// Unfortunately the go-cni library returns interfaces in an unordered map meaning
	// the results may be nondeterministic depending on CNI plugin output so make
	// sure we sort them by interface name.
	names := make([]string, 0, len(res.Interfaces))
	for k := range res.Interfaces {
		names = append(names, k)
	}
	sort.Strings(names)

	// setStatus sets netStatus.Address and netStatus.InterfaceName
	// if it finds a suitable interface that has IP address(es)
	// (at least IPv4, possibly also IPv6)
	setStatus := func(requireSandbox bool) {
		for _, name := range names {
			iface := res.Interfaces[name]
			// this should never happen but this value is coming from external
			// plugins so we should guard against it
			if iface == nil {
				continue
			}

			if requireSandbox && iface.Sandbox == "" {
				continue
			}

			for _, ipConfig := range iface.IPConfigs {
				isIP4 := ipConfig.IP.To4() != nil
				if netStatus.Address == "" && isIP4 {
					netStatus.Address = ipConfig.IP.String()
				}
				if netStatus.AddressIPv6 == "" && !isIP4 {
					netStatus.AddressIPv6 = ipConfig.IP.String()
				}
			}

			// found a good interface (with either IPv4 or IPv6), so we're done
			if netStatus.Address != "" || netStatus.AddressIPv6 != "" {
				netStatus.InterfaceName = name
				return
			}
		}
	}

	// Use the first sandbox interface with an IP address
	setStatus(true)

	// If no IP address was found, use the first interface with an address
	// found as a fallback
	if netStatus.Address == "" && netStatus.AddressIPv6 == "" {
		setStatus(false)
		c.logger.Debug("no sandbox interface with an address found CNI result, using first available",
			"interface", netStatus.InterfaceName,
			"ip", netStatus.Address,
		)
	}

	// If no IP address (IPv4 or IPv6) could be found, return an error
	if netStatus.Address == "" && netStatus.AddressIPv6 == "" {
		return nil, fmt.Errorf("failed to configure network: no interface with an address")

	}

	// Fallback: if no IPv4 address but we have IPv6, copy it to Address field
	// for backward compatibility with code that only checks Address field
	// (e.g. service registration with address_mode="alloc")
	if netStatus.Address == "" && netStatus.AddressIPv6 != "" {
		netStatus.Address = netStatus.AddressIPv6
	}

	// Use the first DNS results, if non-empty
	if len(res.DNS) > 0 {
		cniDNS := res.DNS[0]
		if len(cniDNS.Nameservers) > 0 {
			netStatus.DNS = &structs.DNSConfig{
				Servers:  cniDNS.Nameservers,
				Searches: cniDNS.Search,
				Options:  cniDNS.Options,
			}
		}
	}

	return netStatus, nil
}

// cniConfParser parses different config formats as appropriate
type cniConfParser struct {
	listBytes []byte
	confBytes []byte
}

// getOpt produces a cni.Opt to load with cni.CNI.Load()
func (c *cniConfParser) getOpt() (cni.Opt, error) {
	if len(c.listBytes) > 0 {
		return cni.WithConfListBytes(c.listBytes), nil
	}
	if len(c.confBytes) > 0 {
		return cni.WithConf(c.confBytes), nil
	}
	// theoretically should never be reached
	return nil, errors.New("no CNI network config found")
}

// loadCNIConf looks in confDir for a CNI config with the specified name
func loadCNIConf(confDir, name string) (*cniConfParser, error) {
	files, err := cnilibrary.ConfFiles(confDir, []string{".conf", ".conflist", ".json"})
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to detect CNI config file: %v", err)
	case len(files) == 0:
		return nil, fmt.Errorf("no CNI network config found in %s", confDir)
	}

	// files contains the network config files associated with cni network.
	// Use lexicographical way as a defined order for network config files.
	sort.Strings(files)
	for _, confFile := range files {
		if strings.HasSuffix(confFile, ".conflist") {
			confList, err := cnilibrary.ConfListFromFile(confFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load CNI config list file %s: %v", confFile, err)
			}
			if confList.Name == name {
				return &cniConfParser{
					listBytes: confList.Bytes,
				}, nil
			}
		} else {
			conf, err := cnilibrary.ConfFromFile(confFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load CNI config file %s: %v", confFile, err)
			}
			if conf.Network.Name == name {
				return &cniConfParser{
					confBytes: conf.Bytes,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("CNI network config not found for name %q", name)
}

// Teardown calls the CNI plugins with the delete action
func (c *cniNetworkConfigurator) Teardown(ctx context.Context, alloc *structs.Allocation, spec *drivers.NetworkIsolationSpec) error {
	if err := c.ensureCNIInitialized(); err != nil {
		return err
	}

	portMap := getPortMapping(alloc, c.ignorePortMappingHostIP)

	if err := c.cni.Remove(ctx, alloc.ID, spec.Path, cni.WithCapabilityPortMap(portMap.ports)); err != nil {
		c.logger.Warn("error from cni.Remove; attempting manual iptables cleanup", "err", err)

		// best effort cleanup ipv6
		ipt, iptErr := c.newIPTables(structs.NodeNetworkAF_IPv6)
		if iptErr != nil {
			c.logger.Debug("failed to detect ip6tables", "error", iptErr)
		} else {
			if err := c.forceCleanup(ipt, alloc.ID); err != nil {
				c.logger.Warn("failed to cleanup iptables", "error", err)
			}
		}

		// create a real handle to iptables
		ipt, iptErr = c.newIPTables(structs.NodeNetworkAF_IPv4)
		if iptErr != nil {
			return fmt.Errorf("failed to detect iptables: %w", iptErr)
		}
		// most likely the pause container was removed from underneath nomad
		return c.forceCleanup(ipt, alloc.ID)
	}

	return nil
}

var (
	// ipRuleRe is used to parse a postrouting iptables rule created by nomad, e.g.
	//   -A POSTROUTING -s 172.26.64.191/32 -m comment --comment "name: \"nomad\" id: \"6b235529-8111-4bbe-520b-d639b1d2a94e\"" -j CNI-50e58ea77dc52e0c731e3799
	ipRuleRe = regexp.MustCompile(`-A POSTROUTING -s (\S+) -m comment --comment "name: \\"nomad\\" id: \\"([[:xdigit:]-]+)\\"" -j (CNI-[[:xdigit:]]+)`)
)

// forceCleanup is the backup plan for removing the iptables rule and chain associated with
// an allocation that was using bridge networking. The cni library refuses to handle a
// dirty state - e.g. the pause container is removed out of band, and so we must cleanup
// iptables ourselves to avoid leaking rules.
func (c *cniNetworkConfigurator) forceCleanup(ipt IPTablesCleanup, allocID string) error {
	const (
		natTable         = "nat"
		postRoutingChain = "POSTROUTING"
		commentFmt       = `--comment "name: \"nomad\" id: \"%s\""`
	)

	// list the rules on the POSTROUTING chain of the nat table
	rules, err := ipt.List(natTable, postRoutingChain)
	if err != nil {
		return fmt.Errorf("failed to list iptables rules: %w", err)
	}

	// find the POSTROUTING rule associated with our allocation
	matcher := fmt.Sprintf(commentFmt, allocID)
	var ruleToPurge string
	for _, rule := range rules {
		if strings.Contains(rule, matcher) {
			ruleToPurge = rule
			break
		}
	}

	// no rule found for our allocation, just give up
	if ruleToPurge == "" {
		c.logger.Info("iptables cleanup: did not find postrouting rule for alloc", "alloc_id", allocID)
		return nil
	}

	// re-create the rule we need to delete, as tokens
	subs := ipRuleRe.FindStringSubmatch(ruleToPurge)
	if len(subs) != 4 {
		return fmt.Errorf("failed to parse postrouting rule for alloc %s", allocID)
	}
	cidr := subs[1]
	id := subs[2]
	chainID := subs[3]
	toDel := []string{
		`-s`,
		cidr,
		`-m`,
		`comment`,
		`--comment`,
		`name: "nomad" id: "` + id + `"`,
		`-j`,
		chainID,
	}

	// remove the jump rule
	ok := true
	if err = ipt.Delete(natTable, postRoutingChain, toDel...); err != nil {
		c.logger.Warn("failed to remove iptables nat.POSTROUTING rule", "alloc_id", allocID, "chain", chainID, "error", err)
		ok = false
	}

	// remote the associated chain
	if err = ipt.ClearAndDeleteChain(natTable, chainID); err != nil {
		c.logger.Warn("failed to remove iptables nat chain", "chain", chainID, "error", err)
		ok = false
	}

	if !ok {
		return fmt.Errorf("failed to cleanup iptables rules for alloc %s", allocID)
	}

	return nil
}

func (c *cniNetworkConfigurator) ensureCNIInitialized() error {
	if err := c.cni.Status(); !cni.IsCNINotInitialized(err) {
		return err
	}
	opt, err := c.confParser.getOpt()
	if err != nil {
		return err
	}
	return c.cni.Load(opt)
}

// nsOpts keeps track of NamespaceOpts usage, mainly for test assertions.
type nsOpts struct {
	args  map[string]string
	ports []cni.PortMapping
}

func (o *nsOpts) withArgs(args map[string]string) cni.NamespaceOpts {
	o.args = args
	return cni.WithLabels(args)
}

func (o *nsOpts) withCapabilityPortMap(ports []cni.PortMapping) cni.NamespaceOpts {
	o.ports = ports
	return cni.WithCapabilityPortMap(ports)
}

// portMappings is a wrapper around a slice of cni.PortMapping that lets us
// index via the port's label, which isn't otherwise included in the
// cni.PortMapping struct
type portMappings struct {
	ports  []cni.PortMapping
	labels map[string]int // Label -> index into ports field
}

func (pm *portMappings) set(label string, port cni.PortMapping) {
	pm.ports = append(pm.ports, port)
	pm.labels[label] = len(pm.ports) - 1
}

func (pm *portMappings) get(label string) (cni.PortMapping, bool) {
	idx, ok := pm.labels[label]
	if !ok {
		return cni.PortMapping{}, false
	}
	return pm.ports[idx], true
}

// getPortMapping builds a list of cni.PortMapping structs that are used as the
// portmapping capability arguments for the portmap CNI plugin
func getPortMapping(alloc *structs.Allocation, ignoreHostIP bool) *portMappings {
	mappings := &portMappings{
		ports:  []cni.PortMapping{},
		labels: map[string]int{},
	}

	if len(alloc.AllocatedResources.Shared.Ports) == 0 && len(alloc.AllocatedResources.Shared.Networks) > 0 {
		for _, network := range alloc.AllocatedResources.Shared.Networks {
			for _, port := range append(network.DynamicPorts, network.ReservedPorts...) {
				if port.To < 1 {
					port.To = port.Value
				}
				for _, proto := range []string{"tcp", "udp"} {
					portMapping := cni.PortMapping{
						HostPort:      int32(port.Value),
						ContainerPort: int32(port.To),
						Protocol:      proto,
					}
					mappings.set(port.Label, portMapping)
				}
			}
		}
	} else {
		for _, port := range alloc.AllocatedResources.Shared.Ports {
			if port.To < 1 {
				port.To = port.Value
			}
			for _, proto := range []string{"tcp", "udp"} {

				portMapping := cni.PortMapping{
					HostPort:      int32(port.Value),
					ContainerPort: int32(port.To),
					Protocol:      proto,
				}
				if !ignoreHostIP {
					portMapping.HostIP = port.HostIP
				}
				mappings.set(port.Label, portMapping)
			}
		}
	}
	return mappings
}
