// Package ci provides a small Fluxor CI core for build pipelines.
//
// The package focuses on a simple and extensible model:
//   - Pipeline: ordered stages sharing BuildState
//   - BuildState: artifact registry produced by build stages
//   - RuntimeFromBuild: create runtime image spec by copying artifacts from build state
//
// It is intentionally minimal so apps can plug in their own executors (GitHub Actions,
// GitLab CI, local runner) while reusing a consistent in-process pipeline model.
package ci
