//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	trtesting "github.com/hashicorp/nomad/client/allocrunner/taskrunner/testing"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/taskenv"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
)

func newMinimalTestManager(t *testing.T, templates []*structs.Template) (*TaskTemplateManager, *taskenv.Builder, string) {
	t.Helper()
	alloc := mock.Alloc()
	node := mock.Node()
	builder := taskenv.NewBuilder(node, alloc, alloc.Job.TaskGroups[0].Tasks[0], "global")
	dir := t.TempDir()
	builder.SetClientTaskRoot(dir)
	hooks := trtesting.NewMockTaskHooks()
	mgr, err := NewTaskTemplateManager(&TaskTemplateManagerConfig{
		UnblockCh: hooks.UnblockCh, Lifecycle: hooks, Events: hooks,
		Templates: templates, ClientConfig: &config.Config{}, TaskDir: dir,
		EnvBuilder: builder, NomadNamespace: structs.DefaultNamespace,
		Logger: hclog.NewNullLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return mgr, builder, dir
}

func TestMinimalTemplate_StaticAndEnv(t *testing.T) {
	tmpls := []*structs.Template{
		{EmbeddedTmpl: `hello {{ env "NOMAD_ALLOC_ID" }}`, DestPath: "local/value", Once: true, Perms: "0600"},
		{EmbeddedTmpl: "FROM_TEMPLATE=yes\n", DestPath: "secrets/env", Envvars: true, Once: true, Perms: "0600"},
	}
	mgr, builder, dir := newMinimalTestManager(t, tmpls)
	mgr.Run()
	data, err := os.ReadFile(filepath.Join(dir, "local/value"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "hello ") {
		t.Fatalf("unexpected render: %q", data)
	}
	if got := builder.Build().Map()["FROM_TEMPLATE"]; got != "yes" {
		t.Fatalf("template env = %q", got)
	}
	info, err := os.Stat(filepath.Join(dir, "local/value"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %o", info.Mode().Perm())
	}
}

func TestMinimalTemplate_RejectsConsulAndVaultFunctions(t *testing.T) {
	for _, body := range []string{`{{ key "app/config" }}`, `{{ service "api" }}`, `{{ secret "secret/data/app" }}`} {
		alloc := mock.Alloc()
		builder := taskenv.NewBuilder(mock.Node(), alloc, alloc.Job.TaskGroups[0].Tasks[0], "global")
		builder.SetClientTaskRoot(t.TempDir())
		hooks := trtesting.NewMockTaskHooks()
		_, err := NewTaskTemplateManager(&TaskTemplateManagerConfig{
			UnblockCh: hooks.UnblockCh, Lifecycle: hooks, Events: hooks,
			Templates:    []*structs.Template{{EmbeddedTmpl: body, DestPath: "local/out"}},
			ClientConfig: &config.Config{}, TaskDir: t.TempDir(), EnvBuilder: builder,
			Logger: hclog.NewNullLogger(),
		})
		if err == nil || !strings.Contains(err.Error(), "nomad_min: unsupported template function") {
			t.Fatalf("body %q returned %v", body, err)
		}
	}
}
