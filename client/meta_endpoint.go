// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"maps"
	"net/http"
	"time"

	metrics "github.com/hashicorp/go-metrics"
	"github.com/hashicorp/nomad/nomad/structs"
)

type NodeMeta struct {
	c *Client
}

func newNodeMetaEndpoint(c *Client) *NodeMeta {
	n := &NodeMeta{c: c}
	return n
}

func (n *NodeMeta) Apply(args *structs.NodeMetaApplyRequest, reply *structs.NodeMetaResponse) error {
	defer metrics.MeasureSince([]string{"client", "node_meta", "apply"}, time.Now())

	// Check node write permissions
	if aclObj, err := n.c.ResolveToken(args.AuthToken); err != nil {
		return err
	} else if !aclObj.AllowNodeWrite() {
		return structs.ErrPermissionDenied
	}

	if err := args.Validate(); err != nil {
		return structs.NewErrRPCCoded(http.StatusBadRequest, err.Error())
	}

	// Keep persistence serialized without holding configLock across disk I/O.
	// Readers retain the last durable snapshot until publication below.
	if !n.c.metaWriteLock.TryLock() {
		timer := time.NewTimer(time.Second)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer timer.Stop()
		defer ticker.Stop()
		acquired := false
		for !acquired {
			select {
			case <-timer.C:
				return structs.NewErrRPCCoded(http.StatusServiceUnavailable, "node metadata persistence is busy; retry the update")
			case <-ticker.C:
				acquired = n.c.metaWriteLock.TryLock()
			}
		}
	}
	defer n.c.metaWriteLock.Unlock()

	n.c.configLock.Lock()
	dyn := maps.Clone(n.c.metaDynamic)
	n.c.configLock.Unlock()
	maps.Copy(dyn, args.Meta)
	for key, value := range args.Meta {
		if _, static := n.c.metaStatic[key]; value == nil && !static {
			delete(dyn, key)
		}
	}

	capabilitySaved, healthSaved, metadataSaved := false, false, false
	persist := func() error {
		if args.Capability != nil {
			if err := n.c.stateDB.PutMimirCapability(args.Capability); err != nil {
				return err
			}
			capabilitySaved = true
		}
		if args.Health != nil {
			if err := n.c.stateDB.PutMimirHealth(args.Health); err != nil {
				return err
			}
			healthSaved = true
		}
		if err := n.c.stateDB.PutNodeMeta(dyn); err != nil {
			return err
		}
		metadataSaved = true
		return nil
	}
	stateErr := persist()
	newNode := n.c.UpdateNode(func(node *structs.Node) {
		// Preserve the existing partial-success behavior if a later persistence
		// step fails, but never publish a value whose write failed.
		if capabilitySaved {
			node.MimirCapability = args.Capability.Copy()
			node.Meta["mimir.benchmark_spec"] = args.Capability.BenchmarkSpec
			node.Meta["mimir.benchmark_result"] = args.Capability.ResultID
		}
		if healthSaved {
			node.MimirHealth = args.Health.Copy()
		}
		if metadataSaved {
			n.c.metaDynamic = maps.Clone(dyn)
			for key, value := range args.Meta {
				if value == nil {
					delete(node.Meta, key)
				} else {
					node.Meta[key] = *value
				}
			}
		}
	})
	if stateErr != nil {
		return stateErr
	}

	// Trigger an async node update
	n.c.updateNode()

	reply.Meta = newNode.Meta
	reply.Dynamic = dyn
	reply.Static = n.c.metaStatic
	reply.Capability = newNode.MimirCapability.Copy()
	reply.Health = newNode.MimirHealth.Copy()
	return nil
}

func (n *NodeMeta) Read(args *structs.NodeSpecificRequest, reply *structs.NodeMetaResponse) error {
	defer metrics.MeasureSince([]string{"client", "node_meta", "read"}, time.Now())

	// Check node read permissions
	if aclObj, err := n.c.ResolveToken(args.AuthToken); err != nil {
		return err
	} else if !aclObj.AllowNodeRead() {
		return structs.ErrPermissionDenied
	}

	// Must acquire configLock to ensure reads aren't interleaved with
	// writes
	n.c.configLock.Lock()
	defer n.c.configLock.Unlock()

	reply.Meta = n.c.config.Node.Meta
	reply.Dynamic = maps.Clone(n.c.metaDynamic)
	reply.Static = n.c.metaStatic
	reply.Capability = n.c.config.Node.MimirCapability.Copy()
	reply.Health = n.c.config.Node.MimirHealth.Copy()
	return nil
}
