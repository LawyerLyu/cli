// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/internal/cmdpolicy"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
)

// NewCmdConfigPolicy returns the `config policy` group. Subcommands:
//
//	show              - print the resolved user-layer Rule + source + denied count
//
// The command writes a structured JSON envelope so AI agents and CI
// integrations can parse the result.
func NewCmdConfigPolicy(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "policy",
		Hidden: true, // diagnostic-only; kept callable, omitted from --help to reduce noise
		Short:  "Inspect the user-layer command policy",
		// The parent `config` group has a PersistentPreRunE that calls
		// RequireBuiltinCredentialProvider, which returns external_provider
		// when env credentials are set. `policy show` is a READ-ONLY
		// diagnostic command and does not modify credentials, so it must
		// work regardless of which credential provider is active. A
		// leaf-level no-op PersistentPreRunE wins under cobra's "first
		// walking up" rule and bypasses the parent check.
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			c.SilenceUsage = true
			return nil
		},
	}
	cmd.AddCommand(newCmdConfigPolicyShow(f))
	return cmd
}

func newCmdConfigPolicyShow(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:    "show",
		Hidden: true, // diagnostic-only; kept callable, omitted from --help to reduce noise
		Short:  "Show the active user-layer policy (Plugin.Restrict / yaml / none)",
		Long: `Print the policy currently in effect after bootstrap, including:

  - source: "plugin:<name>" / "yaml" / "none"
  - rule:   the resolved Rule (Allow / Deny / MaxRisk / Identities)
  - yaml_path:    the file location that was examined (informational)
  - yaml_shadowed: true when a plugin Restrict overrides the yaml

A "denied_paths" count reflects the number of commands the engine
marked as denied after father-group aggregation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPolicyShow(f)
		},
	}
}

func runConfigPolicyShow(f *cmdutil.Factory) error {
	active := cmdpolicy.GetActive()
	if active == nil {
		// Bootstrap not yet recorded -- happens when the command is
		// invoked from a context that bypassed buildInternal (only test
		// shells should hit this).
		output.PrintJson(f.IOStreams.Out, map[string]any{
			"source": string(cmdpolicy.SourceNone),
			"note":   "no policy recorded; bootstrap did not run pruning",
		})
		return nil
	}

	out := map[string]any{
		"source":       string(active.Source.Kind),
		"source_name":  active.Source.Name,
		"yaml_path":    active.YAMLPath,
		"denied_paths": active.DeniedPaths,
	}
	if active.Rule != nil {
		out["rule"] = map[string]any{
			"name":              active.Rule.Name,
			"description":       active.Rule.Description,
			"allow":             active.Rule.Allow,
			"deny":              active.Rule.Deny,
			"max_risk":          active.Rule.MaxRisk,
			"identities":        active.Rule.Identities,
			"allow_unannotated": active.Rule.AllowUnannotated,
		}
	}
	// Surface the yaml-shadowed case so a user wondering "why is my
	// yaml ignored?" sees it immediately.
	if active.Source.Kind == cmdpolicy.SourcePlugin && active.YAMLPath != "" {
		if _, err := os.Stat(active.YAMLPath); err == nil {
			out["yaml_shadowed"] = true
			fmt.Fprintln(f.IOStreams.ErrOut,
				"note: a plugin contributed Restrict(); yaml IGNORED")
		}
	}
	output.PrintJson(f.IOStreams.Out, out)
	return nil
}
