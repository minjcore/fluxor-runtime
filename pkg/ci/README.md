# ci (fluxor-ci)

Minimal CI pipeline core for Fluxor packages/apps.

## What it provides

- **Pipeline**: ordered stages sharing one mutable `BuildState`
- **BuildState**: named artifacts produced by build stage
- **RuntimeFromBuild**: create runtime image spec by copying artifact from build state

This models the common CI flow:

1. Build stage produces artifact (`app.jar`, binary, bundle)
2. Runtime stage uses a base runtime image and only copies artifact from build output

## Example

```go
state := ci.NewBuildState()

p := ci.NewPipeline(
    ci.NewFuncStage("build", func(ctx context.Context, s *ci.BuildState) error {
        return s.PutArtifact("jar", "target/app.jar")
    }),
    ci.NewFuncStage("runtime", func(ctx context.Context, s *ci.BuildState) error {
        spec, err := ci.RuntimeFromBuild(s, ci.RuntimeOptions{ArtifactName: "jar"})
        if err != nil {
            return err
        }
        _ = spec // use spec to render Dockerfile/CI job
        return nil
    }),
)

if err := p.Run(context.Background(), state); err != nil {
    panic(err)
}
```

## Defaults in RuntimeFromBuild

- `BaseImage`: `eclipse-temurin:21-jre`
- `TargetPath`: `/app/app.jar`
- `Entrypoint`: `java -jar /app/app.jar`

Override via `RuntimeOptions`.
