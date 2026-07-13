//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package template contains the small, Nomad-native template renderer used by
// nomad_min. It intentionally does not import consul-template or expose Consul
// and Vault template functions.
package template

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	envparse "github.com/hashicorp/go-envparse"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/client/allocrunner/taskrunner/interfaces"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/taskenv"
	"github.com/hashicorp/nomad/nomad/structs"
)

const minimalPollInterval = 2 * time.Second

type TaskTemplateManagerConfig struct {
	UnblockCh      chan struct{}
	Lifecycle      interfaces.TaskLifecycle
	Events         interfaces.EventEmitter
	Templates      []*structs.Template
	ClientConfig   *config.Config
	TaskDir        string
	EnvBuilder     *taskenv.Builder
	NomadNamespace string
	NomadToken     string
	Logger         hclog.Logger
}

func (c *TaskTemplateManagerConfig) Validate() error {
	switch {
	case c == nil:
		return errors.New("nil config passed")
	case c.UnblockCh == nil:
		return errors.New("invalid unblock channel given")
	case c.Lifecycle == nil:
		return errors.New("invalid lifecycle hooks given")
	case c.Events == nil:
		return errors.New("invalid event hook given")
	case c.ClientConfig == nil:
		return errors.New("invalid client config given")
	case c.TaskDir == "":
		return fmt.Errorf("invalid task directory given: %q", c.TaskDir)
	case c.EnvBuilder == nil:
		return errors.New("invalid task environment given")
	}
	return nil
}

type TaskTemplateManager struct {
	config      *TaskTemplateManagerConfig
	stopCh      chan struct{}
	stopOnce    sync.Once
	unblockOnce sync.Once
	apiClient   *api.Client
}

func NewTaskTemplateManager(c *TaskTemplateManagerConfig) (*TaskTemplateManager, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	tm := &TaskTemplateManager{config: c, stopCh: make(chan struct{})}
	for _, tmpl := range c.Templates {
		if tmpl.ChangeMode == structs.TemplateChangeModeSignal && tmpl.ChangeSignal == "" {
			return nil, errors.New("template change_mode signal requires change_signal")
		}
		if err := tm.validateTemplate(tmpl); err != nil {
			return nil, err
		}
	}
	return tm, nil
}

func (tm *TaskTemplateManager) Templates() []*structs.Template { return tm.config.Templates }

func (tm *TaskTemplateManager) Stop() { tm.stopOnce.Do(func() { close(tm.stopCh) }) }

func (tm *TaskTemplateManager) Run() {
	if len(tm.config.Templates) == 0 {
		tm.ready()
		return
	}

	var previous map[*structs.Template][]byte
	for {
		outputs, err := tm.renderAll()
		if err == nil {
			previous = outputs
			tm.ready()
			break
		}
		tm.config.Logger.Warn("minimal template render waiting for dependency", "error", err)
		select {
		case <-tm.stopCh:
			return
		case <-time.After(minimalPollInterval):
		}
	}
	if tm.config.Templates[0].Once {
		return
	}

	ticker := time.NewTicker(minimalPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-tm.stopCh:
			return
		case <-ticker.C:
			outputs, err := tm.renderAll()
			if err != nil {
				tm.config.Logger.Warn("minimal template re-render failed", "error", err)
				continue
			}
			changed := changedTemplates(previous, outputs)
			previous = outputs
			if len(changed) != 0 {
				tm.handleChanges(changed)
			}
		}
	}
}

func (tm *TaskTemplateManager) ready() {
	tm.unblockOnce.Do(func() { close(tm.config.UnblockCh) })
}

func (tm *TaskTemplateManager) validateTemplate(t *structs.Template) error {
	body := t.EmbeddedTmpl
	if t.SourcePath != "" {
		source, escapes := tm.config.EnvBuilder.Build().ClientPath(t.SourcePath, false)
		if escapes {
			return errors.New("template source path escapes alloc directory")
		}
		data, readErr := os.ReadFile(source)
		if readErr != nil {
			return fmt.Errorf("read template source %q: %w", source, readErr)
		}
		body = string(data)
	}
	_, err := tm.parse(t, body)
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, fn := range []string{"key", "keyOrDefault", "keyExists", "service", "services", "secret", "secrets"} {
		if strings.Contains(msg, fmt.Sprintf("function %q not defined", fn)) {
			return fmt.Errorf("nomad_min: unsupported template function %s", fn)
		}
	}
	return fmt.Errorf("nomad_min: invalid template: %w", err)
}

func (tm *TaskTemplateManager) parse(t *structs.Template, body string) (*texttemplate.Template, error) {
	left, right := t.LeftDelim, t.RightDelim
	if left == "" {
		left = "{{"
	}
	if right == "" {
		right = "}}"
	}
	funcs := texttemplate.FuncMap{
		"env":            tm.env,
		"nomadVar":       tm.nomadVar,
		"GetInterfaceIP": getInterfaceIP,
	}
	tpl := texttemplate.New("nomad_min").Funcs(funcs).Delims(left, right)
	if t.ErrMissingKey {
		tpl = tpl.Option("missingkey=error")
	}
	return tpl.Parse(body)
}

func (tm *TaskTemplateManager) renderAll() (map[*structs.Template][]byte, error) {
	outputs := make(map[*structs.Template][]byte, len(tm.config.Templates))
	for _, spec := range tm.config.Templates {
		body := spec.EmbeddedTmpl
		if spec.SourcePath != "" {
			source, escapes := tm.config.EnvBuilder.Build().ClientPath(spec.SourcePath, false)
			if escapes {
				return nil, errors.New("template source path escapes alloc directory")
			}
			b, err := os.ReadFile(source)
			if err != nil {
				return nil, fmt.Errorf("read template source %q: %w", source, err)
			}
			body = string(b)
		}
		tpl, err := tm.parse(spec, body)
		if err != nil {
			return nil, fmt.Errorf("nomad_min: invalid template: %w", err)
		}
		var out bytes.Buffer
		if err := tpl.Execute(&out, nil); err != nil {
			return nil, fmt.Errorf("render template %q: %w", spec.DestPath, err)
		}
		outputs[spec] = out.Bytes()
	}

	for spec, data := range outputs {
		if err := tm.write(spec, data); err != nil {
			return nil, err
		}
	}
	if err := tm.updateEnv(); err != nil {
		return nil, err
	}
	return outputs, nil
}

func (tm *TaskTemplateManager) write(spec *structs.Template, data []byte) error {
	dest, escapes := tm.config.EnvBuilder.Build().ClientPath(spec.DestPath, false)
	if escapes {
		return errors.New("template destination path escapes alloc directory")
	}
	if old, err := os.ReadFile(dest); err == nil && bytes.Equal(old, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if spec.Perms != "" {
		v, err := strconv.ParseUint(spec.Perms, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid template permissions %q: %w", spec.Perms, err)
		}
		perm = os.FileMode(v)
	}
	f, err := os.CreateTemp(filepath.Dir(dest), ".nomad-min-template-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err == nil {
		err = f.Chmod(perm)
	}
	if err == nil && (spec.Uid != nil || spec.Gid != nil) {
		uid, gid := -1, -1
		if spec.Uid != nil {
			uid = *spec.Uid
		}
		if spec.Gid != nil {
			gid = *spec.Gid
		}
		err = f.Chown(uid, gid)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

func (tm *TaskTemplateManager) updateEnv() error {
	all := make(map[string]string)
	taskEnv := tm.config.EnvBuilder.Build()
	for _, spec := range tm.config.Templates {
		if !spec.Envvars {
			continue
		}
		dest, _ := taskEnv.ClientPath(spec.DestPath, true)
		f, err := os.Open(dest)
		if err != nil {
			return fmt.Errorf("open env template: %w", err)
		}
		vars, parseErr := envparse.Parse(f)
		_ = f.Close()
		if parseErr != nil {
			return fmt.Errorf("parse env template %q: %w", dest, parseErr)
		}
		for k, v := range vars {
			all[k] = v
		}
	}
	tm.config.EnvBuilder.SetTemplateEnv(all)
	return nil
}

func (tm *TaskTemplateManager) env(key string) string {
	return tm.config.EnvBuilder.Build().Map()[key]
}

func (tm *TaskTemplateManager) nomadVar(path string) (api.VariableItems, error) {
	client, err := tm.nomadClient()
	if err != nil {
		return nil, err
	}
	items, _, err := client.Variables().GetVariableItems(path, &api.QueryOptions{Namespace: tm.config.NomadNamespace})
	return items, err
}

func (tm *TaskTemplateManager) nomadClient() (*api.Client, error) {
	if tm.apiClient != nil {
		return tm.apiClient, nil
	}
	if tm.config.ClientConfig.TemplateDialer == nil {
		return nil, errors.New("Nomad template API dialer is unavailable")
	}
	transport := &http.Transport{DialContext: tm.config.ClientConfig.TemplateDialer.DialContext}
	c, err := api.NewClient(&api.Config{
		Address:    "http://nomad.internal",
		Namespace:  tm.config.NomadNamespace,
		SecretID:   tm.config.NomadToken,
		HttpClient: &http.Client{Transport: transport},
	})
	if err != nil {
		return nil, err
	}
	tm.apiClient = c
	return c, nil
}

func getInterfaceIP(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	var ipv6 string
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip.To4() != nil {
			return ip.String(), nil
		}
		ipv6 = ip.String()
	}
	if ipv6 != "" {
		return ipv6, nil
	}
	return "", fmt.Errorf("interface %q has no usable IP address", name)
}

func changedTemplates(old, next map[*structs.Template][]byte) []*structs.Template {
	var changed []*structs.Template
	for spec, data := range next {
		if before, ok := old[spec]; !ok || !bytes.Equal(before, data) {
			changed = append(changed, spec)
		}
	}
	return changed
}

func (tm *TaskTemplateManager) handleChanges(changed []*structs.Template) {
	var restart bool
	signals := make(map[string]struct{})
	var scripts []*structs.ChangeScript
	var splay time.Duration
	for _, spec := range changed {
		switch spec.ChangeMode {
		case structs.TemplateChangeModeRestart:
			restart = true
		case structs.TemplateChangeModeSignal:
			signals[spec.ChangeSignal] = struct{}{}
		case structs.TemplateChangeModeScript:
			scripts = append(scripts, spec.ChangeScript)
		}
		if spec.Splay > splay {
			splay = spec.Splay
		}
	}
	if splay > 0 {
		select {
		case <-tm.stopCh:
			return
		case <-time.After(time.Duration(rand.Int63n(int64(splay)))):
		}
	}
	if restart {
		_ = tm.config.Lifecycle.Restart(context.Background(), structs.NewTaskEvent(structs.TaskRestartSignal).SetDisplayMessage("Template with change_mode restart re-rendered"), false)
		return
	}
	for signal := range signals {
		event := structs.NewTaskEvent(structs.TaskSignaling).SetDisplayMessage("Template re-rendered")
		if err := tm.config.Lifecycle.Signal(event, signal); err != nil {
			tm.config.Logger.Error("template signal failed", "signal", signal, "error", err)
		}
	}
	for _, script := range scripts {
		if script == nil {
			continue
		}
		_, code, err := tm.config.Lifecycle.Exec(script.Timeout, script.Command, script.Args)
		if err != nil || code != 0 {
			tm.config.Logger.Error("template change script failed", "exit_code", code, "error", err)
		}
	}
}
