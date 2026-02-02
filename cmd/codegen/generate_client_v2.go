package main

import (
	"fmt"
	"log/slog"

	"dagger.io/dagger"
	"dagger.io/dagger/telemetry"
	"github.com/dagger/dagger/cmd/codegen/generator"
	"github.com/spf13/cobra"
)

var generateClientV2Cmd = &cobra.Command{
	Use:   "generate-client-v2",
	Short: "Generate a client",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
	RunE: GenerateClientV2,
}

func GenerateClientV2(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	ctx = telemetry.InitEmbedded(ctx, nil)
	defer telemetry.Close()

	cfg, err := getGlobalConfig(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get global configuration: %w", err)
	}
	defer cfg.Close()

	clientConfig := &generator.ClientGeneratorConfig{
		ClientDir: outputDir,
	}

	// If a client dir is provided, we use it.
	if clientDir != "" {
		clientConfig.ClientDir = clientDir
	}

	// This should be removed and pass as argument
	moduleSourceID, err := cfg.Dag.ModuleSource(".").ID(ctx)
	if err != nil {
		return fmt.Errorf("failed to load module to generate client for: %w", err)
	}

	if moduleSourceID != "" {
		var res struct {
			Source struct {
				Name          string `json:"moduleOriginalName"`
				EngineVersion string `json:"engineVersion"`
				Dependencies  []generator.ModuleSourceDependency
			}
		}

		err := cfg.Dag.Do(ctx,
			&dagger.Request{
				Query:  loadModuleSourceDepsQuery,
				OpName: "ModuleSourceDependencies",
				Variables: map[string]any{
					"source": dagger.ModuleSourceID(moduleSourceID),
				},
			},
			&dagger.Response{
				Data: &res,
			})
		if err != nil {
			return fmt.Errorf("failed to load module source dependencies: %w", err)
		}

		clientConfig.ModuleName = res.Source.Name
		clientConfig.EngineVersion = res.Source.EngineVersion
		clientConfig.ModuleDependencies = res.Source.Dependencies
	}

	cfg.ClientConfig = clientConfig

	generator, err := getGenerator(cfg)
	if err != nil {
		return fmt.Errorf("failed to get generator: %w", err)
	}

	slog.Info("generating SDK client", "language", cfg.Lang)

	return Generate(ctx, cfg, generator.GenerateClientV2)
}

func init() {
	// Specific client generation flags
	generateClientV2Cmd.Flags().StringVar(&moduleSourceID, "module-source-id", "", "id of the module to generate code for")
	generateClientV2Cmd.Flags().StringVar(&clientDir, "client-dir", "", "directory where the client will be generated (output by default)")
}
