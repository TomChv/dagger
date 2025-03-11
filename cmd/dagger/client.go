package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"

	"dagger.io/dagger"
	"github.com/dagger/dagger/engine/client"
	"github.com/dagger/dagger/engine/client/pathutil"
	"github.com/juju/ansiterm/tabwriter"
	"github.com/spf13/cobra"
)

var (
	generator string
	dev       bool

	clientsListJSONOutput bool
)

func init() {
	clientInstallCmd.Flags().StringVar(&generator, "generator", "", "Generator to use to generate the client")
	clientInstallCmd.Flags().BoolVar(&dev, "dev", false, "Generate in developer mode")

	// Hide `dev` flag since it's only for maintainers.
	_ = clientInstallCmd.Flags().MarkHidden("dev")

	clientListCmd.Flags().BoolVar(&clientsListJSONOutput, "json", false, "output in JSON format")

	clientCmd.AddCommand(clientInstallCmd)
	clientCmd.AddCommand(clientListCmd)
	clientCmd.AddCommand(clientGenerateCmd)
	clientCmd.AddCommand(clientUninstallCmd)
}

var clientCmd = &cobra.Command{
	Use:    "client",
	Short:  "Access Dagger client subcommands",
	Hidden: true,
	Annotations: map[string]string{
		"experimental": "true",
	},
}

var clientInstallCmd = &cobra.Command{
	Use:     "install [options] [path]",
	Short:   "Generate a new Dagger client from the Dagger module",
	Example: "dagger client install --generator=go ./dagger",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withEngine(cmd.Context(), client.Params{}, func(ctx context.Context, engineClient *client.Client) error {
			if generator == "" {
				return fmt.Errorf("generator must set (ts, go, python or custom generator)")
			}

			// default the output to the current working directory if it doesn't exist yet
			cwd, err := pathutil.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			outputPath := filepath.Join(cwd, "dagger")
			if len(args) > 0 {
				outputPath = args[0]
			}

			if filepath.IsAbs(outputPath) {
				outputPath, err = filepath.Rel(cwd, outputPath)
				if err != nil {
					return fmt.Errorf("failed to get relative path: %w", err)
				}
			}

			dag := engineClient.Dagger()

			mod, _, err := initializeClientGeneratorModule(ctx, dag, ".")
			if err != nil {
				return fmt.Errorf("failed to initialize client generator module: %w", err)
			}

			contextDirPath, err := mod.Source.LocalContextDirectoryPath(ctx)
			if err != nil {
				return fmt.Errorf("failed to get local context directory path: %w", err)
			}

			_, err = mod.Source.
				WithClient(generator, outputPath, dagger.ModuleSourceWithClientOpts{
					Dev: dev,
				}).
				GeneratedContextDirectory().
				Export(ctx, contextDirPath)
			if err != nil {
				return fmt.Errorf("failed to export client: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Generated client at %s\n", outputPath)

			return nil
		})
	},
	Annotations: map[string]string{
		"experimental": "true",
	},
}

//go:embed clientconf.graphql
var loadModClientConfQuery string

var clientListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List installed client generators",
	Example: "dagger client list",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withEngine(cmd.Context(), client.Params{}, func(ctx context.Context, engineClient *client.Client) error {
			dag := engineClient.Dagger()

			mod, _, err := initializeClientGeneratorModule(ctx, dag, ".")
			if err != nil {
				return fmt.Errorf("failed to initialize client generator module: %w", err)
			}

			sourceID, err := mod.Source.ID(ctx)
			if err != nil {
				return fmt.Errorf("failed to get module source ID: %w", err)
			}

			var res struct {
				Source struct {
					ConfigClients []struct {
						Generator string
						Directory string
						Dev       bool
					}
				}
			}

			err = dag.Do(ctx, &dagger.Request{
				Query: loadModClientConfQuery,
				Variables: map[string]any{
					"source": sourceID,
				},
			}, &dagger.Response{
				Data: &res,
			})
			if err != nil {
				return fmt.Errorf("failed to query module client config: %w", err)
			}

			if clientsListJSONOutput {
				jsonRes, err := json.Marshal(res.Source.ConfigClients)
				if err != nil {
					return fmt.Errorf("failed to marshal module client config: %w", err)
				}

				cmd.OutOrStdout().Write(jsonRes)
				return nil
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 0, ' ', tabwriter.Debug)
			fmt.Fprintf(tw, "Generator\tDirectory\tDev\n")
			for _, client := range res.Source.ConfigClients {
				fmt.Fprintf(tw, "%s\t%s\t%t\n",
					client.Generator,
					client.Directory,
					client.Dev,
				)
			}

			return tw.Flush()
		})
	},
}

var clientGenerateCmd = &cobra.Command{
	Use:     "generate",
	Short:   "Regenerate clients installed in the current module",
	Example: "dagger client generate",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withEngine(cmd.Context(), client.Params{}, func(ctx context.Context, engineClient *client.Client) error {
			dag := engineClient.Dagger()

			mod, _, err := initializeClientGeneratorModule(ctx, dag, ".")
			if err != nil {
				return fmt.Errorf("failed to initialize client generator module: %w", err)
			}

			contextDirPath, err := mod.Source.LocalContextDirectoryPath(ctx)
			if err != nil {
				return fmt.Errorf("failed to get local context directory path: %w", err)
			}

			_, err = mod.Source.
				GeneratedContextDirectory().
				Export(ctx, contextDirPath)
			if err != nil {
				return fmt.Errorf("failed to export client: %w", err)
			}

			return nil
		})
	},
}

var clientUninstallCmd = &cobra.Command{
	Use:     "uninstall <path>",
	Short:   "Remove a generated client from the module source",
	Example: "dagger client uninstall ./generated",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withEngine(cmd.Context(), client.Params{}, func(ctx context.Context, engineClient *client.Client) error {
			if len(args) != 1 {
				return fmt.Errorf("expected 1 argument, got %d", len(args))
			}

			// default the output to the current working directory if it doesn't exist yet
			cwd, err := pathutil.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			outputPath := filepath.Join(cwd, "dagger")
			if len(args) > 0 {
				outputPath = args[0]
			}

			if filepath.IsAbs(outputPath) {
				outputPath, err = filepath.Rel(cwd, outputPath)
				if err != nil {
					return fmt.Errorf("failed to get relative path: %w", err)
				}
			}

			dag := engineClient.Dagger()

			mod, _, err := initializeClientGeneratorModule(ctx, dag, ".")
			if err != nil {
				return fmt.Errorf("failed to initialize client generator module: %w", err)
			}

			contextDirPath, err := mod.Source.LocalContextDirectoryPath(ctx)
			if err != nil {
				return fmt.Errorf("failed to get local context directory path: %w", err)
			}

			_, err = mod.Source.
				WithoutClient(outputPath).
				GeneratedContextDirectory().
				Export(ctx, contextDirPath)
			if err != nil {
				return fmt.Errorf("failed to export client: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Generated client at %s\n", outputPath)

			return nil
		})
	},
}
