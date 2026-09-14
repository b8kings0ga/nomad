//go:build nomad_min

package taskrunner

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
	"testing"
	"time"
)

func TestMinMimcStatsOptIn(t *testing.T) {
	for _, tc := range []struct {
		driver  string
		publish bool
		want    int
	}{{"mimc", false, 0}, {"mimc", true, 1}, {"docker", true, 0}, {"raw_exec", true, 0}} {
		tr := &TaskRunner{task: &structs.Task{Driver: tc.driver}, clientConfig: &config.Config{PublishAllocationMetrics: tc.publish, StatsCollectionInterval: time.Second}}
		hooks := appendStatsHook(nil, tr, hclog.NewNullLogger())
		if len(hooks) != tc.want {
			t.Fatalf("driver=%s enabled=%v hooks=%d want=%d", tc.driver, tc.publish, len(hooks), tc.want)
		}
		if len(hooks) > 0 {
			h := hooks[0].(*statsHook)
			if h.interval != time.Second || !h.doPublish {
				t.Fatal("existing stats settings not preserved")
			}
		}
	}
}
