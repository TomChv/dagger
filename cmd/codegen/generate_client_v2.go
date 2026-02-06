package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"dagger.io/dagger"
	"dagger.io/dagger/telemetry"
	"github.com/dagger/dagger/cmd/codegen/generator"
	"github.com/dagger/dagger/cmd/codegen/introspection"
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
				Dependencies  []*generator.ModuleSourceDependency
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

	for _, dep := range cfg.ClientConfig.ModuleDependencies {
		depSchema, err := extractModuleSchema(ctx, cfg.Dag, dep.Name, dep.ID)
		if err != nil {
			return err
		}

		dep.Schema = depSchema
	}

	// TODO: remove, it's just for debug
	for _, dep := range cfg.ClientConfig.ModuleDependencies {
		slog.Info("dep schema", "name", dep.Name)

		content, err := json.Marshal(dep.Schema)
		if err != nil {
			return err
		}

		_ = os.WriteFile(fmt.Sprintf("schema-%s.json", dep.Name), []byte(content), 0o600)
	}

	generator, err := getGenerator(cfg)
	if err != nil {
		return fmt.Errorf("failed to get generator: %w", err)
	}

	slog.Info("generating SDK client", "language", cfg.Lang)

	return Generate(ctx, cfg, generator.GenerateClientV2)
}

func extractModuleSchema(ctx context.Context, dag *dagger.Client, moduleName string, moduleID dagger.ModuleSourceID) (*introspection.Schema, error) {
	slog.Info("getting schema for dependency", "name", moduleName, "id", moduleID)

	// TODO: use the introspection schema instead once we use the engine.
	jsonSchema, err := dag.LoadModuleSourceFromID(moduleID).AsModule().JSONSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for dependency: %w", err)
	}

	var introspectionResp introspection.Response
	if err := json.Unmarshal([]byte(jsonSchema), &introspectionResp); err != nil {
		return nil, err
	}

	res := &introspection.Schema{
		QueryType: introspectionResp.Schema.QueryType,
	}

	for _, i := range introspectionResp.Schema.Types {
		if i.Name == "Query" {
			queryType := &introspection.Type{
				Kind:        i.Kind,
				Name:        i.Name,
				Description: i.Description,
				Fields:      []*introspection.Field{},
			}

			for _, field := range i.Fields {
				if field.Directives.Origin() == moduleName {
					queryType.Fields = append(queryType.Fields, field)
				}
			}

			res.Types = append(res.Types, queryType)
		}

		if i.Directives.Origin() == moduleName {
			slog.Info("found type for module", "module", moduleName, "type", i.Name)

			res.Types = append(res.Types, i)
		}
	}
	
	generator.SetSchemaParents(res)

	return res, nil
}

func init() {
	// Specific client generation flags
	generateClientV2Cmd.Flags().StringVar(&moduleSourceID, "module-source-id", "", "id of the module to generate code for")
	generateClientV2Cmd.Flags().StringVar(&clientDir, "client-dir", "", "directory where the client will be generated (output by default)")
}
