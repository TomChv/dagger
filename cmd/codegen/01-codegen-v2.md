# Go Codegen V2 Proposal

## Summary

This proposal introduces a major revision to the Go code generator that unifies how Dagger client code is generated across three distinct use cases: modules, standalone clients, and the core SDK library. The primary goal is to make generated Dagger code truly portable and reusable across different contexts by eliminating code duplication and enabling native Go interoperability.

## Motivation

### Current State and Problems

Today, the Go code generator operates in three different modes, each with its own approach to generating client bindings:

1. **Module generation** (`dagger develop`): Generates code in `internal/dagger/dagger.gen.go` that includes full core API bindings
2. **Standalone client** (`dagger client install`): Generates a separate client with its own copy of core bindings
3. **SDK library** (`sdk/go/dagger.gen.go`): The "official" core bindings published as `dagger.io/dagger`

This fragmentation creates several significant problems:

#### 1. Code Duplication and Maintenance Burden

Every time we generate code for a module or standalone client, we regenerate the entire core API (Container, Directory, File, etc.). This means:

- The same ~10K+ lines of core binding code is duplicated across every module
- Any bug fix or improvement to the core bindings requires regenerating all modules
- The codegen templates are more complex, maintaining multiple paths for the same functionality

#### 2. Zero Interoperability

Because each generated client has its own copy of core type definitions, **code cannot be shared between different projects or packages**. Consider this real-world scenario:

```go
// Project A: A CLI tool with generated client in ./dagger/
// File: project-a/dagger/dagger.gen.go (generated)
package dagger

type Container struct { /* generated code */ }
type Directory struct { /* generated code */ }

// File: project-a/main.go
package main

import "project-a/dagger"

func BuildImage(ctx context.Context) (*dagger.Container, error) {
    dag, err := dagger.Connect(ctx)
    if err != nil {
        return nil, err
    }
    return dag.Container().From("alpine:3.20"), nil
}

// Project B: Another tool that wants to reuse Project A's function
// File: project-b/dagger/client.gen.go (generated)
package dagger

type Container struct { /* SAME generated code, but DIFFERENT type */ }
type Directory struct { /* SAME generated code, but DIFFERENT type */ }

// File: project-b/main.go
package main

import (
    "project-b/dagger"
    projecta "project-a"  // Want to reuse code from Project A
)

func ExtendProjectA(ctx context.Context) (*dagger.Container, error) {
    // This CANNOT work because:
    // - projecta.BuildImage returns project-a/dagger.Container
    // - But this code expects project-b/dagger.Container
    // These are incompatible types even though they're byte-for-byte identical!
    ctr, err := projecta.BuildImage(ctx) // ❌ Type mismatch error
    if err != nil {
        return nil, err
    }
    return ctr.WithExec([]string{"apk", "add", "git"}), nil
}
```

The same problem occurs when trying to:

- Extract common building logic into a shared library
- Build reusable Dagger utilities that work across multiple projects
- Use third-party Go packages that return or accept Dagger types

This violates the fundamental expectation of Go developers that types from the same logical API should be compatible across projects.

#### 3. Cannot Use Dagger-Based Go Libraries

If someone publishes a Go library that accepts `dagger.io/dagger` types:

```go
// Published library: github.com/example/dagger-tools
package tools

import "dagger.io/dagger"

func OptimizeContainer(ctr *dagger.Container) *dagger.Container {
    return ctr.WithExec([]string{"optimize"})
}
```

This library **cannot be used** from within a Dagger module because the module's `internal/dagger.Container` is a different type than `dagger.io/dagger.Container`.

#### 4. Confusing Mental Model

Developers coming to Dagger with Go experience expect:

- Types from `dagger.io/dagger` to be the canonical types
- Code using these types to be portable across projects
- The ability to build libraries and share code like any other Go ecosystem

The current approach breaks all these expectations, creating a steep learning curve and limiting the ecosystem's growth.

### Why This Matters

These issues prevent Dagger from becoming a true Go-native tool. Instead of being able to leverage Go's strengths (type safety, code reuse, ecosystem), we're fighting against them. This limits:

- **Developer productivity**: Can't reuse code patterns across modules
- **Ecosystem growth**: Can't build and share Dagger-aware Go libraries
- **Adoption**: Experienced Go developers find the current model surprising and restrictive
- **Maintenance**: We maintain multiple codepaths for essentially the same functionality

## Goals

The V2 codegen aims to achieve:

### 1. **Unification**: Single Source of Truth

- The official `dagger.io/dagger` package becomes the **only** place where core API types are defined
- All generated code (modules, standalone clients) imports and uses these types instead of regenerating them
- Templates and codegen logic simplified by eliminating multiple code paths

### 2. **Simplification**: Lean Generated Code

- Generated code focuses only on module-specific or dependency-specific APIs
- Core bindings (Container, Directory, etc.) are never regenerated
- Faster generation, smaller diffs, easier reviews

### 3. **Interoperability**: True Go Portability

- Functions accepting `*dagger.Container` can be used anywhere
- Modules can call each other's functions directly with proper type compatibility
- Go libraries can be published that work with Dagger types
- Code can be extracted from modules into shared packages without modification

### Example of Desired State

```go
// Published library
package containerutils

import "dagger.io/dagger"

func AddUser(ctr *dagger.Container, username string) *dagger.Container {
    return ctr.WithExec([]string{"useradd", username})
}

// Module A
package main

import (
    "dagger.io/dagger"
    "github.com/example/containerutils"
)

func (m *ModuleA) BuildImage() *dagger.Container {
    return containerutils.AddUser(
        dag.Container().From("alpine"),
        "app",
    )
}
```

## Overall Implementation Plan

The migration to V2 involves several coordinated changes across the codegen system:

### Phase 1: New Client Generation Architecture

**Goal**: Establish the foundation where generated clients depend on `dagger.io/dagger` instead of embedding core bindings.

**Key Changes**:

1. **Separate Client Module Structure**
   - Generate client code in a dedicated subdirectory (e.g., `dagger/`)
   - Each client gets its own `go.mod` that requires `dagger.io/dagger`
   - Parent module uses `replace` directive to reference the generated client locally

2. **Template Restructuring**
   - Split templates into:
     - Core bindings (only for `sdk/go/dagger.gen.go`)
     - Client-specific code (type aliases, dependency APIs, utilities)
   - Client templates use type aliases: `type Container = dagger.Container`
   - Remove all core API generation from module/client templates

3. **Dependency Resolution**
   - Pin `dagger.io/dagger` version based on `dagger.json` engine version
   - Handle development versions with appropriate replace directives
   - Ensure `go mod tidy` works correctly for both parent and client modules

### Phase 2: Module Generation Updates

**Goal**: Adapt `dagger develop` to use the new client generation approach.

**Key Changes**:

1. **Module Structure**
   - Keep module source in root directory
   - Generate client bindings in `dagger/` subdirectory with its own `go.mod`
   - Module imports client as: `import "example.com/mymodule/dagger"`

2. **DAG Access**
   - The `dag` global remains available through the generated client package
   - Module code uses: `dagger.Container`, `dagger.Directory`, etc.
   - Generated `dag` variable provides access to: `dag.Container()`, `dag.Git()`, module dependencies

3. **Backward Compatibility**
   - Provide migration path for existing modules
   - Consider temporary support for old structure during transition
   - Clear error messages and migration documentation

### Phase 3: Engine Schema Metadata Enhancements

**Goal**: Enable codegen to distinguish between core API types and module-specific types using schema introspection.

**Current State**:

Currently, the engine's GraphQL schema doesn't expose metadata about which module defined each type. When codegen introspects the schema, it receives a flat list of all types (core API types like `Container`, `Directory` mixed with module-defined types) without any way to distinguish their origin.

This makes it impossible for codegen to:

- Know which types are part of the core API and should be imported from `dagger.io/dagger`
- Identify which types come from which dependencies
- Generate appropriate code organization based on type ownership

**Required Changes**:

1. **Add `@origin` directive to track type ownership**

   Introduce a new GraphQL directive that annotates types with their originating module:

   ```graphql
   """
   Indicates the module that defined this type.
   Core API types have no @origin directive.
   """
   directive @origin(module: String!) on OBJECT | INTERFACE | ENUM | SCALAR
   ```

   **Semantics**:
   - No `@origin` directive → Core API type (Container, Directory, etc.)
   - `@origin(module: "hello")` → Type defined by "hello" module

2. **Apply directive during type registration**

   When types are registered, attach the `@origin` directive to their definitions:

   ```go
   // dagql/types.go - For ID types
   func (i ID[T]) TypeDefinition(view call.View) *ast.Definition {
       typeDef := &ast.Definition{
           Kind: ast.Scalar,
           Name: i.TypeName(),
           // ...
       }

       if i.origin != "" {
           typeDef.Directives = append(typeDef.Directives, DirectiveOrigin(i.origin))
       }

       return typeDef
   }

   // dagql/server.go - For object types
   func (s *Server) InstallObject(class ObjectType) ObjectType {
       // ... existing code

       if class.Origin() != "" {
           // Apply @origin directive to the type definition
           spec.Directives = append(spec.Directives, DirectiveOrigin(class.Origin()))
       }

       // ...
   }
   ```

3. **Set origin when installing module types**

   Module types are created with their origin set to the module name:

   ```go
   // When a module installs its types
   func (m *Module) Install(ctx context.Context, srv *dagql.Server) error {
       for _, typeDef := range m.TypeDefs() {
           class := dagql.NewClass[*MyType](srv, dagql.ClassOpts[*MyType]{
               Origin: m.Name(), // Set module name as origin
           })
           srv.InstallObject(class)
       }
       return nil
   }
   ```

   Core types are installed **without** an origin (empty string), which means they won't have the `@origin` directive.

4. **Schema partitioning in codegen**

   When generating a client, the codegen needs to categorize types into three buckets:

   ```go
   // Core types (no @origin directive)
   // → Import from dagger.io/dagger, create type aliases
   type Container = dagger.Container
   type Directory = dagger.Directory

   // Dependency types (@origin(module: "hello"))
   // → Generate as new types in dependency package
   package hello
   type Hello struct { /* ... */ }

   // Query root extensions
   // → Generate accessors on dag object
   func (q *Query) Hello() *hello.Hello { /* ... */ }
   ```

### Phase 4: Dynamic Core Type Extensions

**Goal**: Handle core types that are dynamically extended based on installed dependencies.

**The Problem**:

Certain core types like `Env` and `Binding` are **not static** - they're dynamically extended at runtime based on what types are available in the schema. This creates a challenge for the V2 approach where we want to use `dagger.io/dagger` as a static import.

**Current Behavior**:

The engine uses `EnvHook.InstallObject()` to automatically extend `Env` and `Binding` types whenever a new object type is registered:

```go
// For each installed object type (e.g., Container, Directory, Hello)
// Env gets extended with:
envType.Extend(
    FieldSpec{
        Name: "withContainerInput",  // or withHelloInput, etc.
        Type: envType.Typed(),
        // ...
    },
    // ...
)

envType.Extend(
    FieldSpec{
        Name: "withContainerOutput",  // or withHelloOutput, etc.
        Type: envType.Typed(),
        // ...
    },
    // ...
)

// Binding gets extended with:
bindingType.Extend(
    FieldSpec{
        Name: "asContainer",  // or asHello, etc.
        Type: targetType.Typed(),
        // ...
    },
    // ...
)
```

This means:

- Core types: `Env.withContainerInput()`, `Env.withDirectoryInput()`, etc. are in `dagger.io/dagger`
- Module types: `Env.withHelloInput()`, `Binding.asHello()` are added when "hello" module is installed

**The Challenge for V2**:

If we make `Env` and `Binding` pure type aliases to `dagger.io/dagger`, we lose the ability to add module-specific methods like `withHelloInput()` or `asHello()`.

```go
// In generated client
type Env = dagger.Env        // This is a type alias
type Binding = dagger.Binding

// Problem: How do we add withHelloInput() to dagger.Env?
// We can't extend an aliased type!
```

**Proposed Solutions**:

**Option 1: Generate Extension Methods on Wrapper Types**

Instead of pure type aliases, generate thin wrapper types that embed the core type and add extension methods:

```go
// In generated client
type Env struct {
    *dagger.Env  // Embed core type
}

// Generated methods for dependencies
func (e *Env) WithHelloInput(name string, value *Hello, description string) *Env {
    // Implementation that calls underlying methods
}

func (e *Env) WithHelloOutput(name string, description string) *Env {
    // Implementation
}

type Binding struct {
    *dagger.Binding
}

func (b *Binding) AsHello() (*Hello, error) {
    // Implementation
}
```

| Pros | Cons |
|------|------|
| Clean separation: core methods from `dagger.Env`, extensions generated | Not pure type aliases - slight API surface difference |
| Type compatibility: `*client.Env` can be converted to `*dagger.Env` easily | Requires conversion methods or careful handling at API boundaries |
| Familiar pattern: embedding is idiomatic Go | Generated types are slightly "heavier" |

**Option 2: Dynamic Method Resolution (Advanced)**

Use interface-based design where the client provides a richer implementation:

```go
// In dagger.io/dagger
type Env interface {
    WithInput(name string, value any, description string) Env
    WithOutput(name string, typeName string, description string) Env
    // ... core methods
}

// In generated client
type EnvImpl struct {
    *dagger.EnvImpl  // Concrete implementation from SDK
}

// Type-safe generated wrappers
func (e *EnvImpl) WithHelloInput(name string, value *Hello, description string) *EnvImpl {
    return e.WithInput(name, value, description).(*EnvImpl)
}
```

| Pros | Cons |
|------|------|
| Maximum flexibility | Requires significant SDK refactoring |
| Clear separation of concerns | More complex implementation |
| Could enable future architectural improvements | May impact performance (interface dispatch) |

**Recommendation**:

**Start with Option 1 (Wrapper Types)** as it provides the best balance of:

- Maintaining type safety and ergonomics
- Clear code generation patterns
- Reasonable compatibility story

The wrapper approach is a well-understood Go pattern and makes it explicit that the generated client is a "rich" version of the core types with dependency-specific extensions.

**Implementation Notes**:

1. Template logic needs to detect dynamically-extended types (Env, Binding)
2. Generate wrapper structs for these specific types only
3. Provide helper functions for wrapping/unwrapping when crossing API boundaries

**Example Generated Code**:

```go
// client.gen.go
package dagger

import (
    "dagger.io/dagger"
    "myproject/dagger/hello"  // dependency
)

// Env wraps dagger.Env with dependency-specific methods
type Env struct {
    *dagger.Env
}

// Wrap converts dagger.Env to client Env
func WrapEnv(env *dagger.Env) *Env {
    return &Env{Env: env}
}

// Unwrap extracts the underlying dagger.Env
func (e *Env) Unwrap() *dagger.Env {
    return e.Env
}

// Generated: extension for Hello module
func (e *Env) WithHelloInput(name string, value *hello.Hello, description string) *Env {
    // Call underlying WithInput with type-checking
    return WrapEnv(e.Env.WithInput(name, value, description))
}
```

### Phase 5: Type System and Marshaling Challenges

**Goal**: Solve the fundamental portability issues around JSON marshaling, context handling, and optional arguments that prevent true code reuse.

**The Core Problem**:

Currently, generated types rely on **module-specific global variables** for marshaling and query building:

```go
// Generated in each module's internal/dagger package
var (
    dag        *Query  // Module-specific instance
    marshalCtx context.Context  // Module-specific context
)

// Every generated type uses these globals
func (c *Container) UnmarshalJSON(data []byte) error {
    // Uses module-specific 'dag' to load types
    return dag.LoadContainerFromID(id)
}

func (c *Container) MarshalJSON() ([]byte, error) {
    // Uses module-specific 'marshalCtx'
    return json.Marshal(c.ID(marshalCtx))
}
```

This creates **hard module dependencies** that make code non-portable. Even if we use `dagger.io/dagger` types, the generated code still references module-specific globals.

**Challenge 1: Marshaling Context Dependency**

**Problem**: Types need context for marshaling but can't rely on package-level variables in a portable world.

**Current State**:
```go
// In module A
var marshalCtx context.Context

type Container struct { /* ... */ }

func (c *Container) MarshalJSON() ([]byte, error) {
    // Reads from module A's marshalCtx
    return json.Marshal(c.ID(marshalCtx))
}
```

If this code is imported by module B, it still references module A's `marshalCtx`, breaking portability.

**Proposed Solutions**:

**Option A: Use encoding/json.v2 with Context-Aware Unmarshaling**

Leverage the upcoming `encoding/json/v2` package that supports passing context:

```go
// In dagger.io/dagger
func (c *Container) UnmarshalJSONV2(ctx jsonv2.UnmarshalContext, dec *jsonv2.Decoder) error {
    // Context is available via ctx
    dagClient := ctx.Value(dagClientKey).(*Client)
    return dagClient.LoadContainerFromID(id)
}
```

| Pros | Cons |
|------|------|
| Clean solution once json.v2 is stable | json.v2 is not yet released/stable |
| Standard Go approach | Requires waiting for Go ecosystem adoption |
| No API breaking changes needed | Transition period complexity |

**Option B: Dependency Injection Pattern**

Use a factory/builder pattern where context is injected:

```go
// In dagger.io/dagger
type MarshalContext struct {
    Ctx context.Context
    Dag *Client
}

func (c *Container) WithMarshalContext(mc *MarshalContext) *ContainerMarshaler {
    return &ContainerMarshaler{container: c, ctx: mc}
}

type ContainerMarshaler struct {
    container *Container
    ctx       *MarshalContext
}

func (cm *ContainerMarshaler) MarshalJSON() ([]byte, error) {
    return json.Marshal(cm.container.ID(cm.ctx.Ctx))
}
```

| Pros | Cons |
|------|------|
| No breaking changes to core API | More complex API surface |
| Portable and explicit | Wrapper types add cognitive overhead |
| Works with current Go version | Marshaling becomes multi-step |

**Challenge 2: Optional Argument Structs**

**Problem**: Methods with optional arguments use option structs that are generated per-module, breaking portability.

**Current State**:
```go
// Generated in module A's internal/dagger
type ContainerWithExecOpts struct {
    Stdin                          string
    RedirectStdout                 string
    RedirectStderr                 string
    ExperimentalPrivilegedNesting  bool
    InsecureRootCapabilities       bool
}

func (c *Container) WithExec(args []string, opts ...ContainerWithExecOpts) *Container {
    // ...
}
```

If module B wants to call module A's function, it can't construct `ContainerWithExecOpts` because it's defined in module A's internal package.

**Why Interfaces Don't Work**:

A proposed solution was to use interfaces:

```go
type ContainerWithExecOptsInterface interface {
    GetStdin() string
    GetRedirectStdout() string
    // ...
}
```

But this fails because:
1. **Partial initialization is lost**: Structs allow `opts := ContainerWithExecOpts{Stdin: "foo"}`, leaving other fields zero-valued. Interfaces require implementing all methods.
2. **Verbose construction**: Would require building a full implementation for every call
3. **Breaking change**: Signature changes from `*dagger.Container` to `dagger.Container`

**Proposed Solutions**:

Instead of generating in `dagger.io/dagger`, generate option structs in each client:

```go
// In generated client package
type ContainerWithExecOpts = dagger.ContainerWithExecOpts  // If dagger.io/dagger exports it

// Or generate a wrapper if not exported
type ContainerWithExecOpts struct {
    Stdin string
    // ... mirror all fields from dagger.ContainerWithExecOpts
}

func (opts *ContainerWithExecOpts) toDagger() *dagger.ContainerWithExecOpts {
    return &dagger.ContainerWithExecOpts{
        Stdin: opts.Stdin,
        // ... copy all fields
    }
}
```

| Pros | Cons |
|------|------|
| Maintains current ergonomic API | Duplication if not using type aliases |
| No breaking changes | Conversion overhead if wrapping |
| Partial initialization still works | Still somewhat coupled to SDK structure |

**Challenge 3: Query Builder Integration**

**Problem**: Type aliases must properly implement `querybuilder.GraphQLMarshaller` and maintain query building semantics.

```go
// Current generated type
type Container struct {
    q *querybuilder.Selection
}

func (c *Container) WithExec(args []string) *Container {
    q := c.q.Select("withExec").Arg("args", args)
    return &Container{q: q}
}
```

With type aliases (`type Container = dagger.Container`), the implementation is in `dagger.io/dagger`, but generated clients may need to intercept or modify queries.

**Proposed Solution**:

Keep query building entirely in `dagger.io/dagger`. Generated clients only add:
1. Type aliases for core types
2. Dependency-specific types and methods
3. Connection/initialization code

The query builder integration remains transparent - aliased types automatically get the implementation from SDK.

## Technical Considerations

### Go Module Mechanics

The V2 approach relies on proper Go module structure:

```
mymodule/
├── go.mod                    # Parent module
├── main.go                   # Module source
├── dagger/
│   ├── go.mod               # Generated client module
│   ├── client.gen.go        # Client utilities, connection management
│   ├── core.gen.go          # Type aliases: Container = dagger.Container
│   └── [dependency].gen.go  # Generated dependency APIs (e.g., dag.Hello())
└── dagger.json
```

The parent `go.mod` includes:

```go
require mymodule/dagger v0.0.0
replace mymodule/dagger => ./dagger
```

This allows the module to use generated types while the generator maintains the client code separately.

### Version Compatibility

Critical consideration: ensuring generated clients are compatible with the engine version.

**Approach**:

- Use `dagger.json` `engineVersion` to pin `dagger.io/dagger` version
- Validate client version matches engine version at runtime
- Clear error messages for version mismatches

### Performance Implications

**Improvements Expected**:

- Faster generation (no core API regeneration)
- Smaller git diffs (only module code changes)
- Allow generated files to be commited.
- Faster module loading (less code to compile)

**Potential Concerns**:

- Additional `go.mod` resolution overhead (minimal)
- Need to ensure `go mod tidy` doesn't slow things down

### Breaking Changes

V2 introduces breaking changes for existing modules:

1. **Import paths change**: `internal/dagger` → module-specific client package
2. **Package structure**: Client generated in subdirectory
3. **Type definitions**: Using type aliases instead of generated types

**Mitigation**: Clear migration path, tooling support, and comprehensive documentation.

## Success Metrics

We'll know V2 is successful when:

- [ ] Modules can share code by importing each other's packages
- [ ] A Go library using `dagger.io/dagger` can be published and used in modules
- [ ] Generated code size reduced by >80% (no core duplication)
- [ ] Codegen performance improved
- [ ] Community starts building and sharing Dagger-aware Go libraries

## Next Steps

1. Review and refine this proposal with team feedback
2. Build proof-of-concept for Phase 1 (client generation)
3. Validate approach with real-world module examples
4. Address open questions through experimentation

## Related Work

- [**PR #11547**](https://github.com/dagger/dagger/pull/11547): Initial exploration of portable APIs with `--portable-api` flag
- [**Issue #11417**](https://github.com/dagger/dagger/issues/11417): Original request for reusable Go module code
- [**Issue 9878**](https://github.com/dagger/dagger/issues/9878): Introspection API idea
