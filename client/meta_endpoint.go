// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"maps"
	"net/http"
	"time"

	metrics "github.com/hashicorp/go-metrics/compat"
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

	var stateErr error
	var dyn map[string]*string

	newNode := n.c.UpdateNode(func(node *structs.Node) {
		if args.Capability != nil {
			if stateErr = n.c.stateDB.PutMimirCapability(args.Capability); stateErr != nil {
				return
			}
			node.MimirCapability = args.Capability.Copy()
			node.Meta["mimir.benchmark_spec"] = args.Capability.BenchmarkSpec
			node.Meta["mimir.benchmark_result"] = args.Capability.ResultID
		}
		if args.Health != nil {
			if stateErr = n.c.stateDB.PutMimirHealth(args.Health); stateErr != nil {
				return
			}
			node.MimirHealth = args.Health.Copy()
		}
		// First update the Client's state store. This must be done
		// atomically with updating the metadata inmemory to avoid
		// bad interleaving between concurrent updates.
		dyn = maps.Clone(n.c.metaDynamic)
		maps.Copy(dyn, args.Meta)

		// Delete null values from the dynamic metadata if they are also not
		// static. Static null values must be kept so their removal is
		// persisted in client state.
		for k, v := range args.Meta {
			_, static := n.c.metaStatic[k]
			if v == nil && !static {
				delete(dyn, k)
			}
		}

		if stateErr = n.c.stateDB.PutNodeMeta(dyn); stateErr != nil {
			return
		}

		// Apply updated dynamic metadata to client and node now that the part of
		// the operation that can fail succeeded (persistence). Must clone as dyn
		// is read outside of UpdateNode.
		n.c.metaDynamic = maps.Clone(dyn)

		for k, v := range args.Meta {
			if v == nil {
				delete(node.Meta, k)
				continue
			}

			node.Meta[k] = *v
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
