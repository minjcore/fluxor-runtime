package ci

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestBuildState_PutAndGetArtifact(t *testing.T) {
	state := NewBuildState()
	if err := state.PutArtifact("jar", "target/app.jar"); err != nil {
		t.Fatalf("PutArtifact: %v", err)
	}
	got, ok := state.Artifact("jar")
	if !ok {
		t.Fatal("artifact not found")
	}
	if got.Path != "target/app.jar" {
		t.Errorf("artifact path: got %q, want %q", got.Path, "target/app.jar")
	}
}

func TestBuildState_PutArtifact_Validation(t *testing.T) {
	state := NewBuildState()
	if err := state.PutArtifact("", "x"); err == nil {
		t.Error("expected error for empty name")
	}
	if err := state.PutArtifact("x", ""); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestPipeline_Run_OrderedStages(t *testing.T) {
	state := NewBuildState()
	order := make([]string, 0, 2)
	p := NewPipeline(
		NewFuncStage("build", func(ctx context.Context, state *BuildState) error {
			order = append(order, "build")
			return state.PutArtifact("jar", "target/app.jar")
		}),
		NewFuncStage("runtime", func(ctx context.Context, state *BuildState) error {
			order = append(order, "runtime")
			_, err := RuntimeFromBuild(state, RuntimeOptions{ArtifactName: "jar"})
			return err
		}),
	)
	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"build", "runtime"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("stage order: got %v, want %v", order, want)
	}
}

func TestPipeline_Run_StopsOnError(t *testing.T) {
	state := NewBuildState()
	boom := errors.New("boom")
	runtimeExecuted := false
	p := NewPipeline(
		NewFuncStage("build", func(ctx context.Context, state *BuildState) error {
			return boom
		}),
		NewFuncStage("runtime", func(ctx context.Context, state *BuildState) error {
			runtimeExecuted = true
			return nil
		}),
	)
	err := p.Run(context.Background(), state)
	if err == nil {
		t.Fatal("expected error")
	}
	if runtimeExecuted {
		t.Error("runtime stage should not execute after build failure")
	}
}

func TestRuntimeFromBuild_Defaults(t *testing.T) {
	state := NewBuildState()
	_ = state.PutArtifact("jar", "target/app.jar")
	spec, err := RuntimeFromBuild(state, RuntimeOptions{ArtifactName: "jar"})
	if err != nil {
		t.Fatalf("RuntimeFromBuild: %v", err)
	}
	if spec.BaseImage != "eclipse-temurin:21-jre" {
		t.Errorf("BaseImage: got %q", spec.BaseImage)
	}
	if len(spec.Copies) != 1 || spec.Copies[0].FromArtifact != "jar" || spec.Copies[0].ToPath != "/app/app.jar" {
		t.Errorf("unexpected copies: %+v", spec.Copies)
	}
	wantEntry := []string{"java", "-jar", "/app/app.jar"}
	if !reflect.DeepEqual(spec.Entrypoint, wantEntry) {
		t.Errorf("Entrypoint: got %v, want %v", spec.Entrypoint, wantEntry)
	}
}

func TestRuntimeFromBuild_ArtifactMissing(t *testing.T) {
	state := NewBuildState()
	_, err := RuntimeFromBuild(state, RuntimeOptions{ArtifactName: "missing"})
	if !errors.Is(err, ErrArtifactNotFound) {
		t.Errorf("expected ErrArtifactNotFound, got %v", err)
	}
}
