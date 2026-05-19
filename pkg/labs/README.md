// EventBus → jsoniter (hoặc sonic khi Go 1.24+ fix)
// HTTP APIs → encoding/json hoặc jsoniter
// Config Loading → encoding/json (đơn giản)
// Large Payloads → jsoniter hoặc sonic (tương lai)
# Labs Package - Experimental Core Features

The `labs` package is **"Core v2"** - a sandbox for experimental core features that helps prevent `pkg/core` from becoming bloated. This package contains core-like features that are:

- 🔬 **Experimental**: Under active development and testing
- ⚠️ **Unstable**: APIs may change without notice
- 🚀 **Innovative**: Testing new core patterns, architectures, and ideas
- 🎯 **Future Core Features**: Potential candidates for promotion to `pkg/core`

## Purpose: Keep Core Lean

The `labs` package serves as:

1. **Experimental Core**: Test new core features before adding to `pkg/core`
2. **Core v2**: Alternative core implementations and patterns
3. **Core Extension**: Experimental extensions to existing core functionality
4. **Core Refactoring**: Test major refactorings of core concepts before committing
5. **Keep Core Small**: Prevent `pkg/core` from becoming bloated with experimental code

## Labs vs Core

### `pkg/core` (Stable Core)
- ✅ **Stable, production-ready** core functionality
- ✅ **Base classes**: BaseVerticle, BaseService, BaseHandler, etc.
- ✅ **Concurrency primitives**: EventBus, GoCMD, WorkerPool, Reactor
- ✅ **Well-tested** and documented
- ✅ **API stability** guaranteed
- ✅ **Core patterns** that are proven

### `pkg/labs` (Experimental Core)
- 🔬 **Experimental** core features being tested
- 🔬 **New base classes** or patterns
- 🔬 **Core extensions** or alternatives
- 🔬 **Major refactorings** of core concepts
- ⚠️ **No stability guarantees**
- 🎯 **Future candidates** for `pkg/core`

### Workflow: Labs → Core

```
New Core Idea → pkg/labs/feature → Test & Iterate → pkg/core/feature
     ↓              (experimental)                    (stable)
```

1. **Start in Labs**: New core features start in `pkg/labs`
2. **Test & Iterate**: Experiment, get feedback, refine
3. **Promote to Core**: When stable, move to `pkg/core`
4. **Keep Core Lean**: Only stable, proven features in `pkg/core`

## Guidelines

### When to Use Labs (Core-like Features)

- ✅ **Experimental core features**: New base classes, concurrency primitives, event loop patterns
- ✅ **Core extensions**: Experimental extensions to BaseVerticle, EventBus, GoCMD, etc.
- ✅ **Core alternatives**: Alternative implementations of core concepts
- ✅ **Core refactoring**: Major refactorings of core functionality being tested
- ✅ **New core patterns**: New patterns that might become core patterns
- ✅ **Core utilities**: Experimental utilities that might become core utilities
- ✅ **Breaking core changes**: Breaking changes to core concepts being tested

### When NOT to Use Labs

- ❌ Stable, production-ready features (use dedicated packages like `pkg/web`, `pkg/workflow`, etc.)
- ❌ **Stable core functionality** (use `pkg/core` - only stable core goes here)
- ❌ Well-established patterns (use appropriate stable packages)
- ❌ Non-core features (use appropriate domain packages)

## Current Experiments

### Available Experiments

- **[JSON Library Benchmarks](./jsons/)** - Comprehensive benchmark tests for comparing JSON libraries (encoding/json, jsoniter, gjson, etc.) with performance measurements, memory allocation analysis, and use case recommendations. See [jsons package README](./jsons/README.md) for details.

### Research & Analysis Documents

Technical research and comparison documents that inform future core decisions:

- **[JSON Library Comparison](./json_libs_comparison.md)** - Comprehensive comparison of Go JSON libraries (encoding/json, jsoniter, sonic, easyjson, etc.) with benchmarks, features, and recommendations for high-performance scenarios.

### Proposed Core-like Experiments

These are experimental **core features** that might eventually move to `pkg/core`:

- [ ] **Advanced Base Classes**: New base class patterns (BaseActor, BaseStream, etc.)
- [ ] **Enhanced Event Loop**: Advanced event loop patterns and optimizations
- [ ] **New Concurrency Primitives**: Experimental concurrency utilities
- [ ] **Core Extensions**: Extensions to BaseVerticle, EventBus, GoCMD
- [ ] **Alternative EventBus**: Experimental EventBus implementations
- [ ] **Core Utilities**: Experimental core utilities (JSON, validation, etc.)
- [ ] **Core Patterns**: New core patterns being tested
- [ ] **Core Refactoring**: Major core refactorings being tested

### Non-Core Experiments (Use Other Packages)

For non-core experiments, use appropriate packages:
- **GraphQL**: `pkg/web` or new `pkg/graphql` package
- **gRPC Streaming**: `pkg/connectors/grpc` or new package
- **WebAssembly**: `pkg/wasm` (already exists)
- **Event Sourcing**: New `pkg/eventsourcing` package
- **CQRS**: New `pkg/cqrs` package

## Contributing

### Adding an Experiment

1. Create a new subdirectory or file for your experiment
2. Add clear documentation about:
   - What it does
   - Why it's experimental
   - Known limitations
   - Usage examples
3. Follow Fluxor conventions:
   - Use base classes when available
   - Non-blocking operations
   - Event-driven patterns
   - Fail-fast validation

### Example Structure

```
pkg/labs/
├── README.md           # This file
├── graphql/            # GraphQL experiment
│   ├── README.md
│   ├── server.go
│   └── client.go
├── wasm/               # WebAssembly experiment
│   ├── README.md
│   └── worker.go
└── eventsourcing/      # Event sourcing experiment
    ├── README.md
    └── store.go
```

## Promotion to Stable

When an experiment is ready for production:

1. **Stability**: Feature is stable and well-tested
2. **Documentation**: Complete documentation and examples
3. **API Design**: Finalized API that won't change
4. **Community Feedback**: Positive feedback from users
5. **Migration Path**: Clear migration path from labs to stable

Then:
- **For core features**: Move to `pkg/core` (this is the primary promotion path for labs)
- **For non-core features**: Move to appropriate stable package (e.g., `pkg/web`, `pkg/workflow`, etc.)
- Update documentation
- Add deprecation notice in labs (if needed)
- Announce in changelog

### Promotion to Core

When a **core-like experiment** is ready:
1. Move from `pkg/labs/feature` → `pkg/core/feature` (or integrate into existing core)
2. Ensure full test coverage
3. Update all documentation
4. Add to core exports
5. Announce breaking changes (if any)

## Versioning

Labs experiments follow semantic versioning but with a **0.x** prefix to indicate experimental status:

- `v0.1.0`: Initial experiment
- `v0.2.0`: Breaking changes
- `v0.9.0`: Near-stable, preparing for promotion

## Stability Guarantees

**⚠️ NO STABILITY GUARANTEES**

- APIs may change at any time
- Features may be removed without notice
- Breaking changes are expected
- Use at your own risk

## Examples

### Using a Labs Experiment

```go
import "github.com/fluxorio/fluxor/pkg/labs/graphql"

// Use experimental GraphQL server
server := graphql.NewServer()
// ... experimental API ...
```

### Contributing a Core-like Experiment

```go
package labs

import "github.com/fluxorio/fluxor/pkg/core"

// BaseActor is an experimental base class for actor pattern
// This is a candidate for promotion to pkg/core when stable
type BaseActor struct {
    *core.BaseVerticle
    // Experimental actor implementation
}

// NewBaseActor creates a new experimental base actor
// WARNING: This API is experimental and may change
// When stable, this will move to pkg/core
func NewBaseActor(name string) *BaseActor {
    return &BaseActor{
        BaseVerticle: core.NewBaseVerticle(name),
    }
}

// This follows core patterns: non-blocking, event-driven, fail-fast
func (a *BaseActor) doStart(ctx core.FluxorContext) error {
    // Experimental core implementation
    return nil
}
```

## License

Same as Fluxor Framework - see main LICENSE file.

## Contact

For questions or contributions to labs experiments:
- Open an issue on GitHub
- Discuss in Fluxor community channels
- Contact maintainers

---

**Remember**: Labs is for experimentation. Use stable packages for production! 🚀
