// Package labs is "Core v2" - a sandbox for experimental core features
// that helps prevent pkg/core from becoming bloated.
//
// This package contains core-like features that are:
//   - Experimental: Under active development and testing
//   - Unstable: APIs may change without notice
//   - Core-focused: Experimental core patterns, base classes, and utilities
//   - Future Core: Potential candidates for promotion to pkg/core
//
// WARNING: This package has NO stability guarantees. APIs may change at any time,
// features may be removed without notice, and breaking changes are expected.
// Use at your own risk.
//
// The labs package serves as:
//   - Experimental Core: Test new core features before adding to pkg/core
//   - Core v2: Alternative core implementations and patterns
//   - Core Extension: Experimental extensions to existing core functionality
//   - Core Refactoring: Test major refactorings of core concepts
//   - Keep Core Small: Prevent pkg/core from becoming bloated
//
// When a core-like experiment is ready for production, it will be moved to
// pkg/core (the primary promotion path) with proper documentation and
// stability guarantees. Non-core experiments move to appropriate domain packages.
//
// See README.md for more information about contributing core experiments.
package labs
