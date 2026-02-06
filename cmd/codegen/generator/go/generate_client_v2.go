package gogenerator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dagger/dagger/cmd/codegen/generator"
	"github.com/dagger/dagger/cmd/codegen/introspection"
	"github.com/dschmidt/go-layerfs"
	"github.com/psanford/memfs"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

const goModFilename = "go.mod"

func (g *GoGenerator) GenerateClientV2(ctx context.Context, schema *introspection.Schema, schemaVersion string) (*generator.GeneratedState, error) {
	generator.SetSchema(schema)
	clientConfig := g.Config.ClientConfig

	slog.Info("generating client v2", "outputDir", g.Config.OutputDir, "clientDir", clientConfig.ClientDir)
	slog.Info("module information", "name", clientConfig.ModuleName, "engine version", clientConfig.EngineVersion)

	mfs := memfs.New()

	// Create the directory structure for the client dir
	if err := mfs.MkdirAll(filepath.Clean(clientConfig.ClientDir), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create the client directory: %w", err)
	}

	// Get the project go.mod
	parentGoMod, parentModFound, err := readGoMod(g.Config.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project go.mod: %w", err)
	}

	if !parentModFound {
		// TODO: What should we do in this case? For now I'll assume there's always a go.mod
		// at the project root
		slog.Warn("no parent go.mod found")
		return &generator.GeneratedState{}, nil
	}

	parentModName := parentGoMod.Module.Mod.Path
	clientModName := filepath.Join(parentModName, clientConfig.ClientDir)
	slog.Info("Go project found", "parent module name", parentModName, "client module name", clientModName)

	clientDirAbsPath := filepath.Join(g.Config.OutputDir, clientConfig.ClientDir)
	clientPkgInfo, err := g.getOrCreateClientGoMod(clientModName, clientDirAbsPath, mfs)
	if err != nil {
		return nil, err
	}

	slog.Info("generating files", "package import", clientPkgInfo.PackageName, "package name", clientPkgInfo.PackageImport)

	if err := generateClient(ctx, g.Config, schema, schemaVersion, mfs, &PackageInfo{
		PackageName:   clientPkgInfo.PackageName,
		PackageImport: clientPkgInfo.PackageImport,
	}, nil, nil, 1); err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	layers := []fs.FS{mfs}

	return &generator.GeneratedState{
		Overlay: layerfs.New(layers...),
	}, nil
}

func (g *GoGenerator) getOrCreateClientGoMod(clientModName string, clientDir string, mfs *memfs.FS) (*PackageInfo, error) {
	clientConfig := g.Config.ClientConfig

	clientGoMod, clientModFound, err := readGoMod(clientDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read client go.mod: %w", err)
	}

	clientPkgName := "dagger"
	clientPkgImport := clientModName
	if !clientModFound {
		slog.Info("client go.mod not found, creating one...")

		clientGoMod = new(modfile.File)
		clientGoMod.AddModuleStmt(clientModName)
		clientGoMod.AddGoStmt(goVersion)
	} else {
		slog.Info("client go.mod found, the generated client will use its import path")

		clientPkgImport = clientGoMod.Module.Mod.Path
	}

	// If the version of that module is not a dev version, we can pull the corresponding
	// library from the registry. Otherwise, we let `go mod tidy` resolve the dependency.
	if !isDevVersion(clientConfig.EngineVersion) {
		slog.Info("setting client dagger.io/dagger package", "version", clientConfig.EngineVersion)
		clientGoMod.AddRequire("dagger.io/dagger", clientConfig.EngineVersion)
	}

	// Update or generate the go.mod file on the host.
	modBody, err := clientGoMod.Format()
	if err != nil {
		return nil, fmt.Errorf("failed to format go.mod: %w", err)
	}

	clientGoModPath := filepath.Join(clientDir, goModFilename)
	if err := mfs.WriteFile(clientGoModPath, modBody, 0600); err != nil {
		return nil, fmt.Errorf("failed to create client go.mod: %w", err)
	}

	return &PackageInfo{
		PackageName:   clientPkgName,
		PackageImport: clientPkgImport,
	}, nil
}

func isDevVersion(version string) bool {
	if version == "" {
		return true
	}

	return strings.Contains(semver.Prerelease(version), "-dev-")
}

// Read the go.mod at the given path.
//
// If found, return the go.mod parsed, otherwise return false.
func readGoMod(dirPath string) (*modfile.File, bool, error) {
	goModFile, err := os.ReadFile(filepath.Join(dirPath, goModFilename))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to read go.mod: %w", err)
	}

	goMod, err := modfile.Parse("go.mod", goModFile, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	return goMod, true, nil
}
