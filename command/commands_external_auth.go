// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package command

import "github.com/hashicorp/cli"

func registerExternalAuthCommands(commands map[string]cli.CommandFactory, meta Meta) {
	commands["acl auth-method"] = func() (cli.Command, error) { return &ACLAuthMethodCommand{Meta: meta}, nil }
	commands["acl auth-method create"] = func() (cli.Command, error) { return &ACLAuthMethodCreateCommand{Meta: meta}, nil }
	commands["acl auth-method delete"] = func() (cli.Command, error) { return &ACLAuthMethodDeleteCommand{Meta: meta}, nil }
	commands["acl auth-method info"] = func() (cli.Command, error) { return &ACLAuthMethodInfoCommand{Meta: meta}, nil }
	commands["acl auth-method list"] = func() (cli.Command, error) { return &ACLAuthMethodListCommand{Meta: meta}, nil }
	commands["acl auth-method update"] = func() (cli.Command, error) { return &ACLAuthMethodUpdateCommand{Meta: meta}, nil }
	commands["acl binding-rule"] = func() (cli.Command, error) { return &ACLBindingRuleCommand{Meta: meta}, nil }
	commands["acl binding-rule create"] = func() (cli.Command, error) { return &ACLBindingRuleCreateCommand{Meta: meta}, nil }
	commands["acl binding-rule delete"] = func() (cli.Command, error) { return &ACLBindingRuleDeleteCommand{Meta: meta}, nil }
	commands["acl binding-rule info"] = func() (cli.Command, error) { return &ACLBindingRuleInfoCommand{Meta: meta}, nil }
	commands["acl binding-rule list"] = func() (cli.Command, error) { return &ACLBindingRuleListCommand{Meta: meta}, nil }
	commands["acl binding-rule update"] = func() (cli.Command, error) { return &ACLBindingRuleUpdateCommand{Meta: meta}, nil }
	commands["login"] = func() (cli.Command, error) { return &LoginCommand{Meta: meta}, nil }
}
