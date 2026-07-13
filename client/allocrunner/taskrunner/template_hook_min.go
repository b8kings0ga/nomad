//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import (
	"context"
	"fmt"
	"sync"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	ti "github.com/hashicorp/nomad/client/allocrunner/taskrunner/interfaces"
	"github.com/hashicorp/nomad/client/allocrunner/taskrunner/template"
	"github.com/hashicorp/nomad/client/config"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/client/taskenv"
	"github.com/hashicorp/nomad/nomad/structs"
)

const templateHookName = "template"

type templateHookConfig struct {
	alloc               *structs.Allocation
	logger              log.Logger
	lifecycle           ti.TaskLifecycle
	events              ti.EventEmitter
	templates           []*structs.Template
	clientConfig        *config.Config
	envBuilder          *taskenv.Builder
	consulNamespace     string
	nomadNamespace      string
	renderOnTaskRestart bool
	hookResources       *cstructs.AllocHookResources
}

type templateHook struct {
	config          *templateHookConfig
	logger          log.Logger
	templateManager *template.TaskTemplateManager
	managerLock     sync.Mutex
	nomadToken      string
	taskDir         string
}

func newTemplateHook(config *templateHookConfig) *templateHook {
	return &templateHook{config: config, logger: config.logger.Named(templateHookName)}
}

func (*templateHook) Name() string { return templateHookName }

func (h *templateHook) Prestart(ctx context.Context, req *interfaces.TaskPrestartRequest, _ *interfaces.TaskPrestartResponse) error {
	h.managerLock.Lock()
	defer h.managerLock.Unlock()

	if req.Task.Consul != nil || req.Task.Vault != nil {
		return fmt.Errorf("nomad_min: unsupported template integration consul/vault")
	}
	if h.templateManager != nil {
		if !h.config.renderOnTaskRestart {
			return nil
		}
		h.templateManager.Stop()
		h.templateManager = nil
	}

	h.taskDir = req.TaskDir.Dir
	h.nomadToken = req.NomadToken
	once, watch := []*structs.Template{}, []*structs.Template{}
	for _, tmpl := range h.config.templates {
		if tmpl.Once {
			once = append(once, tmpl)
		} else {
			watch = append(watch, tmpl)
		}
	}
	return h.renderTemplates(ctx, once, watch)
}

func (h *templateHook) newManager(tmpls []*structs.Template) (*template.TaskTemplateManager, chan struct{}, error) {
	unblock := make(chan struct{})
	m, err := template.NewTaskTemplateManager(&template.TaskTemplateManagerConfig{
		UnblockCh:      unblock,
		Lifecycle:      h.config.lifecycle,
		Events:         h.config.events,
		Templates:      tmpls,
		ClientConfig:   h.config.clientConfig,
		TaskDir:        h.taskDir,
		EnvBuilder:     h.config.envBuilder,
		NomadNamespace: h.config.nomadNamespace,
		NomadToken:     h.nomadToken,
		Logger:         h.logger,
	})
	return m, unblock, err
}

func (h *templateHook) renderTemplates(ctx context.Context, once, watch []*structs.Template) error {
	onceMgr, onceReady, err := h.newManager(once)
	if err != nil {
		return err
	}
	watchMgr, watchReady, err := h.newManager(watch)
	if err != nil {
		return err
	}
	go onceMgr.Run()
	go watchMgr.Run()
	select {
	case <-ctx.Done():
		onceMgr.Stop()
		return ctx.Err()
	case <-onceReady:
	}
	select {
	case <-ctx.Done():
		watchMgr.Stop()
		return ctx.Err()
	case <-watchReady:
	}
	h.templateManager = watchMgr
	return nil
}

func (h *templateHook) Stop(context.Context, *interfaces.TaskStopRequest, *interfaces.TaskStopResponse) error {
	h.managerLock.Lock()
	defer h.managerLock.Unlock()
	if h.templateManager != nil {
		h.templateManager.Stop()
	}
	return nil
}

func (h *templateHook) Update(ctx context.Context, req *interfaces.TaskUpdateRequest, _ *interfaces.TaskUpdateResponse) error {
	h.managerLock.Lock()
	defer h.managerLock.Unlock()
	if h.templateManager == nil || req.NomadToken == h.nomadToken {
		return nil
	}
	tmpls := h.templateManager.Templates()
	h.templateManager.Stop()
	h.templateManager = nil
	h.nomadToken = req.NomadToken
	return h.renderTemplates(ctx, nil, tmpls)
}
