// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"errors"
	"maps"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/state"
	"github.com/hashicorp/nomad/nomad/structs"
)

type delayedMetaDB struct {
	state.NoopDB
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
	saved   map[string]*string
	failure error
}

func (s *delayedMetaDB) PutNodeMeta(meta map[string]*string) error {
	if s.calls.Add(1) == 1 && s.entered != nil {
		close(s.entered)
	}
	if s.release != nil {
		<-s.release
	}
	if s.failure != nil {
		return s.failure
	}
	s.saved = maps.Clone(meta)
	return nil
}
func metaTestClient(db state.StateDB) *Client {
	return &Client{stateDB: db, config: &config.Config{Node: &structs.Node{ID: "test", Meta: map[string]string{"static": "keep"}, Attributes: map[string]string{}}}, metaStatic: map[string]string{"static": "keep"}, metaDynamic: map[string]*string{}}
}

func TestNodeMetaSlowPersistenceDoesNotBlockReaders(t *testing.T) {
	db := &delayedMetaDB{entered: make(chan struct{}), release: make(chan struct{})}
	c := metaTestClient(db)
	endpoint := newNodeMetaEndpoint(c)
	done := make(chan error, 1)
	go func() {
		done <- endpoint.Apply(&structs.NodeMetaApplyRequest{Meta: map[string]*string{"first": new("durable")}}, &structs.NodeMetaResponse{})
	}()
	<-db.entered
	released := false
	defer func() {
		if !released {
			close(db.release)
		}
	}()
	read := make(chan *structs.NodeMetaResponse, 1)
	go func() {
		r := &structs.NodeMetaResponse{}
		if err := endpoint.Read(&structs.NodeSpecificRequest{}, r); err != nil {
			read <- nil
			return
		}
		read <- r
	}()
	select {
	case snapshot := <-read:
		if snapshot == nil || snapshot.Meta["static"] != "keep" || snapshot.Meta["first"] != "" {
			t.Fatal("reader did not retain the last durable snapshot")
		}
	case <-time.After(time.Second):
		t.Fatal("metadata I/O blocked a config reader")
	}
	// Heartbeat/fingerprint updates must remain live and survive publication.
	c.UpdateNode(func(n *structs.Node) { n.Attributes["concurrent"] = "preserve" })
	start := time.Now()
	err := endpoint.Apply(&structs.NodeMetaApplyRequest{Meta: map[string]*string{"second": new("retry")}}, &structs.NodeMetaResponse{})
	if err == nil || !strings.Contains(err.Error(), "persistence is busy") || time.Since(start) > 2*time.Second {
		t.Fatalf("unbounded admission: %v", err)
	}
	if db.calls.Load() != 1 {
		t.Fatal("rejected update reached persistence")
	}
	close(db.release)
	released = true
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := c.Node(); got.Meta["first"] != "durable" || got.Attributes["concurrent"] != "preserve" {
		t.Fatal("publication lost a committed update")
	}
	if *db.saved["first"] != "durable" {
		t.Fatal("memory was published without persistence")
	}
	if err := endpoint.Apply(&structs.NodeMetaApplyRequest{Meta: map[string]*string{"second": new("retry")}}, &structs.NodeMetaResponse{}); err != nil {
		t.Fatal(err)
	}
	if c.Node().Meta["first"] != "durable" || c.Node().Meta["second"] != "retry" {
		t.Fatal("retry lost prior metadata")
	}
}

func TestNodeMetaFailedPersistenceDoesNotPublish(t *testing.T) {
	db := &delayedMetaDB{failure: errors.New("write failed")}
	c := metaTestClient(db)
	err := newNodeMetaEndpoint(c).Apply(&structs.NodeMetaApplyRequest{Meta: map[string]*string{"uncommitted": new("bad")}}, &structs.NodeMetaResponse{})
	if err == nil || c.Node().Meta["uncommitted"] != "" || len(c.metaDynamic) != 0 {
		t.Fatal("failed write was published")
	}
}
