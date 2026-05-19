package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/fluxorio/fluxor/pkg/ci"
)

type config struct {
	artifactName string
	artifactPath string
	baseImage    string
	targetPath   string
	entrypoint   string
	printDocker  bool
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.artifactName, "artifact-name", "jar", "artifact name in build state")
	flag.StringVar(&cfg.artifactPath, "artifact-path", "target/upstream-service-springboot-0.0.1-SNAPSHOT.jar", "artifact path produced by build stage")
	flag.StringVar(&cfg.baseImage, "base-image", "eclipse-temurin:21-jre", "runtime base image")
	flag.StringVar(&cfg.targetPath, "target-path", "/app/app.jar", "target path in runtime image")
	flag.StringVar(&cfg.entrypoint, "entrypoint", "", "comma-separated entrypoint, e.g. java,-jar,/app/app.jar")
	flag.BoolVar(&cfg.printDocker, "print-dockerfile", true, "print generated runtime-only Dockerfile snippet")
	flag.Parse()
	return cfg
}

func splitEntrypoint(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runtimeDockerfile(spec ci.RuntimeSpec, fromArtifact, artifactPath string) string {
	entry := make([]string, 0, len(spec.Entrypoint))
	for _, e := range spec.Entrypoint {
		entry = append(entry, fmt.Sprintf("%q", e))
	}
	return fmt.Sprintf("FROM %s\nWORKDIR /app\n# copy from build artifact registry\n# %s -> %s (%s)\nCOPY %s %s\nENTRYPOINT [%s]\n",
		spec.BaseImage,
		fromArtifact,
		spec.Copies[0].ToPath,
		artifactPath,
		artifactPath,
		spec.Copies[0].ToPath,
		strings.Join(entry, ", "),
	)
}

func main() {
	cfg := parseFlags()
	state := ci.NewBuildState()

	pipeline := ci.NewPipeline(
		ci.NewFuncStage("build", func(ctx context.Context, s *ci.BuildState) error {
			// Simulate build output artifact in CI.
			return s.PutArtifact(cfg.artifactName, cfg.artifactPath)
		}),
		ci.NewFuncStage("runtime", func(ctx context.Context, s *ci.BuildState) error {
			spec, err := ci.RuntimeFromBuild(s, ci.RuntimeOptions{
				ArtifactName: cfg.artifactName,
				BaseImage:    cfg.baseImage,
				TargetPath:   cfg.targetPath,
				Entrypoint:   splitEntrypoint(cfg.entrypoint),
			})
			if err != nil {
				return err
			}

			fmt.Println("fluxor-ci firsttest")
			fmt.Printf("base image: %s\n", spec.BaseImage)
			fmt.Printf("copy: %s -> %s\n", spec.Copies[0].FromArtifact, spec.Copies[0].ToPath)
			fmt.Printf("entrypoint: %v\n", spec.Entrypoint)
			if cfg.printDocker {
				fmt.Println("\n--- runtime-only Dockerfile snippet ---")
				fmt.Print(runtimeDockerfile(spec, cfg.artifactName, cfg.artifactPath))
			}
			return nil
		}),
	)

	if err := pipeline.Run(context.Background(), state); err != nil {
		log.Fatalf("pipeline failed: %v", err)
	}
}

