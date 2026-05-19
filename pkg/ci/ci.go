package ci

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

var (
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrNilBuildState    = errors.New("build state is nil")
)

// Artifact represents an output produced by build stages (binary, jar, bundle, etc.).
type Artifact struct {
	Name string
	Path string
}

// BuildState is shared state across pipeline stages.
// It stores named artifacts produced by earlier stages.
type BuildState struct {
	mu        sync.RWMutex
	artifacts map[string]Artifact
}

// NewBuildState creates an empty state.
func NewBuildState() *BuildState {
	return &BuildState{artifacts: map[string]Artifact{}}
}

// PutArtifact registers/overwrites a named artifact path.
func (s *BuildState) PutArtifact(name, path string) error {
	if s == nil {
		return ErrNilBuildState
	}
	if name == "" {
		return fmt.Errorf("artifact name cannot be empty")
	}
	if path == "" {
		return fmt.Errorf("artifact path cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artifacts[name] = Artifact{Name: name, Path: path}
	return nil
}

// Artifact returns an artifact by name.
func (s *BuildState) Artifact(name string) (Artifact, bool) {
	if s == nil {
		return Artifact{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.artifacts[name]
	return a, ok
}

// Stage is one step in a pipeline.
type Stage interface {
	Name() string
	Run(ctx context.Context, state *BuildState) error
}

// FuncStage adapts a function into a Stage.
type FuncStage struct {
	name string
	fn   func(ctx context.Context, state *BuildState) error
}

// NewFuncStage creates a stage from fn.
func NewFuncStage(name string, fn func(ctx context.Context, state *BuildState) error) *FuncStage {
	failfast.NotEmpty(name, "stage name")
	failfast.NotNil(fn, "stage fn")
	return &FuncStage{name: name, fn: fn}
}

func (s *FuncStage) Name() string { return s.name }

func (s *FuncStage) Run(ctx context.Context, state *BuildState) error {
	failfast.NotNil(ctx, "context")
	if state == nil {
		return ErrNilBuildState
	}
	return s.fn(ctx, state)
}

// Pipeline executes stages in order.
type Pipeline struct {
	stages []Stage
}

// NewPipeline builds a pipeline from ordered stages.
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Run executes all stages in order, stopping at first error.
func (p *Pipeline) Run(ctx context.Context, state *BuildState) error {
	failfast.NotNil(ctx, "context")
	if state == nil {
		return ErrNilBuildState
	}
	for _, stage := range p.stages {
		if stage == nil {
			return fmt.Errorf("nil stage in pipeline")
		}
		if err := stage.Run(ctx, state); err != nil {
			return fmt.Errorf("stage %q failed: %w", stage.Name(), err)
		}
	}
	return nil
}

// RuntimeCopySpec describes one artifact copy operation from build state to runtime image.
type RuntimeCopySpec struct {
	FromArtifact string
	ToPath       string
}

// RuntimeSpec describes a minimal runtime image produced from build output.
type RuntimeSpec struct {
	BaseImage  string
	Entrypoint []string
	Copies     []RuntimeCopySpec
}

// RuntimeOptions configures RuntimeFromBuild.
type RuntimeOptions struct {
	BaseImage    string
	ArtifactName string
	TargetPath   string
	Entrypoint   []string
}

// RuntimeFromBuild creates a runtime image spec that copies one artifact from build state.
// This models the common CI pattern: build stage outputs artifact, runtime stage only copies it.
func RuntimeFromBuild(state *BuildState, opts RuntimeOptions) (RuntimeSpec, error) {
	if state == nil {
		return RuntimeSpec{}, ErrNilBuildState
	}
	if opts.ArtifactName == "" {
		return RuntimeSpec{}, fmt.Errorf("artifact name is required")
	}
	if _, ok := state.Artifact(opts.ArtifactName); !ok {
		return RuntimeSpec{}, fmt.Errorf("%w: %s", ErrArtifactNotFound, opts.ArtifactName)
	}
	base := opts.BaseImage
	if base == "" {
		base = "eclipse-temurin:21-jre"
	}
	target := opts.TargetPath
	if target == "" {
		target = "/app/app.jar"
	}
	entrypoint := opts.Entrypoint
	if len(entrypoint) == 0 {
		entrypoint = []string{"java", "-jar", target}
	}

	return RuntimeSpec{
		BaseImage:  base,
		Entrypoint: entrypoint,
		Copies: []RuntimeCopySpec{{
			FromArtifact: opts.ArtifactName,
			ToPath:       target,
		}},
	}, nil
}
