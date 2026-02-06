package gogenerator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/psanford/memfs"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"

	"github.com/dagger/dagger/cmd/codegen/generator"
	"github.com/dagger/dagger/cmd/codegen/generator/go/templates"
	"github.com/dagger/dagger/cmd/codegen/introspection"
)

const (
	// ClientGenFile is the path to write the codegen for the dagger API
	ClientGenFile = "dagger.gen.go"

	// StarterTemplateFile is the path to write the default module code
	StarterTemplateFile = "main.go"
)

var goVersion = strings.TrimPrefix(runtime.Version(), "go")

type GoGenerator struct {
	Config generator.Config
}

// Sort template keys for deterministic processing
func sortTmplKeys(tmpls map[string]*template.Template) []string {
	keys := make([]string, 0, len(tmpls))
	for k := range tmpls {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	return keys
}

func keepCoreSchemaOnly(schema *introspection.Schema) *introspection.Schema {
	res := &introspection.Schema{
		QueryType: schema.QueryType,
	}

	for _, i := range schema.Types {
		if i.Name == "Query" {
			queryType := &introspection.Type{
				Kind:        i.Kind,
				Name:        i.Name,
				Description: i.Description,
				Fields:      []*introspection.Field{},
			}

			for _, field := range i.Fields {
				if field.Directives.Origin() == "" {
					queryType.Fields = append(queryType.Fields, field)
				}
			}

			res.Types = append(res.Types, queryType)
		} else {
			if i.Directives.Origin() == "" {
				res.Types = append(res.Types, i)
			}
		}
	}

	content, err := json.Marshal(res)
	if err == nil {
		_ = os.WriteFile("core.schema.json", []byte(content), 0o644)
	}

	return res
}

func generateClient(
	ctx context.Context,
	cfg generator.Config,
	schema *introspection.Schema,
	schemaVersion string,
	mfs *memfs.FS,
	pkgInfo *PackageInfo,
	pkg *packages.Package,
	fset *token.FileSet,
	pass int,
) error {
	clientConfig := cfg.ClientConfig

	for _, dep := range clientConfig.ModuleDependencies {
		depFuncs := templates.GoTemplateFuncs(ctx, dep.Schema, schemaVersion, cfg, pkg, fset, pass)
		depTmpl := templates.ClientDependencyTemplates(dep.Name, depFuncs)
		filename := fmt.Sprintf("%s.gen.go", dep.Name)

		slog.Info("generating dependency template", "name", dep.Name, "filename", filename)
		dt, err := renderFile(clientConfig.ClientDir, dep.Schema, schemaVersion, pkgInfo, depTmpl)
		if err != nil {
			return err
		}
		if dt == nil {
			continue
		}

		if err := mfs.WriteFile(filepath.Join(clientConfig.ClientDir, filename), dt, 0600); err != nil {
			return err
		}
	}

	// Generate the root client files.
	coreSchema := keepCoreSchemaOnly(schema)
	funcs := templates.GoTemplateFuncs(ctx, coreSchema, schemaVersion, cfg, pkg, fset, pass)
	tmpls := templates.ClientRootTemplates(funcs)

	for _, k := range sortTmplKeys(tmpls) {
		tmpl := tmpls[k]
		dt, err := renderFile(clientConfig.ClientDir, coreSchema, schemaVersion, pkgInfo, tmpl)
		if err != nil {
			return err
		}
		if dt == nil {
			continue
		}

		if err := mfs.MkdirAll(filepath.Join(clientConfig.ClientDir, filepath.Dir(k)), 0o755); err != nil {
			return err
		}
		if err := mfs.WriteFile(filepath.Join(clientConfig.ClientDir, k), dt, 0600); err != nil {
			return err
		}
	}

	return nil
}

func generateCode(
	ctx context.Context,
	cfg generator.Config,
	schema *introspection.Schema,
	schemaVersion string,
	mfs *memfs.FS,
	pkgInfo *PackageInfo,
	pkg *packages.Package,
	fset *token.FileSet,
	pass int,
) error {
	funcs := templates.GoTemplateFuncs(ctx, schema, schemaVersion, cfg, pkg, fset, pass)
	tmpls := templates.Templates(funcs)

	// Sort template keys for deterministic processing
	keys := make([]string, 0, len(tmpls))
	for k := range tmpls {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		tmpl := tmpls[k]
		dt, err := renderFile(cfg.OutputDir, schema, schemaVersion, pkgInfo, tmpl)
		if err != nil {
			return err
		}
		if dt == nil {
			// no contents, skip
			continue
		}

		// Special case for client generation, we want to write the file in the specified client directory.
		if cfg.ClientConfig != nil && cfg.ClientConfig.ClientDir != "" {
			if err := mfs.MkdirAll(filepath.Join(cfg.ClientConfig.ClientDir, filepath.Dir(k)), 0o755); err != nil {
				return err
			}
			if err := mfs.WriteFile(filepath.Join(cfg.ClientConfig.ClientDir, k), dt, 0600); err != nil {
				return err
			}

			continue
		}

		if err := mfs.MkdirAll(filepath.Dir(k), 0o755); err != nil {
			return err
		}
		if err := mfs.WriteFile(k, dt, 0600); err != nil {
			return err
		}
	}

	return nil
}

func renderFile(
	outputDir string,
	schema *introspection.Schema,
	schemaVersion string,
	pkgInfo *PackageInfo,
	tmpl *template.Template,
) ([]byte, error) {
	data := struct {
		*PackageInfo
		Schema        *introspection.Schema
		SchemaVersion string
		Types         []*introspection.Type
	}{
		PackageInfo:   pkgInfo,
		Schema:        schema,
		SchemaVersion: schemaVersion,
		Types:         schema.Visit(),
	}

	var render bytes.Buffer
	if err := tmpl.Execute(&render, data); err != nil {
		fmt.Printf("partial: %s\n", string(render.Bytes()))
		return nil, err
	}

	source := render.Bytes()
	source = bytes.TrimSpace(source)
	if len(source) == 0 {
		return nil, nil
	}

	formatted, err := format.Source(source)
	if err != nil {
		os.Stderr.Write(source)
		return nil, fmt.Errorf("error formatting generated code: %w", err)
	}
	formatted, err = imports.Process(filepath.Join(outputDir, "dummy.go"), formatted, nil)
	if err != nil {
		os.Stderr.Write(source)
		return nil, fmt.Errorf("error processing imports in generated code: %w", err)
	}
	return formatted, nil
}
