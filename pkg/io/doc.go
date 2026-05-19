// Package io provides I/O utilities for Fluxor.
//
// This package extends the standard library io package with Fluxor-specific
// utilities for file operations, streaming, buffering, and data transformation.
// All functions follow Fluxor's fail-fast principles and provide clear error messages.
//
// Example usage:
//
//	// Read file content
//	data, err := io.ReadFile("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Write file with permissions
//	err = io.WriteFile("output.json", data, 0644)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Copy with progress tracking
//	progress := func(bytes int64) {
//	    fmt.Printf("Copied %d bytes\n", bytes)
//	}
//	err = io.CopyWithProgress(dst, src, progress)
//
//	// Read JSON file
//	var config Config
//	err = io.ReadJSON("config.json", &config)
//
//	// Write JSON file
//	err = io.WriteJSON("output.json", config, 0644)
//
// Features:
//   - File operations: ReadFile, WriteFile, CopyFile, MoveFile, DeleteFile
//   - Streaming: Copy, CopyN, CopyWithProgress, TeeReader
//   - Buffering: Buffer, BufferPool, ReadAll
//   - Structured data: ReadJSON, WriteJSON, ReadYAML, WriteYAML
//   - Reader utilities: LimitReader, MultiReader, ReadAtLeast
//   - Writer utilities: MultiWriter, WriteString, WriteByte
//   - Path utilities: Exists, IsDir, IsFile, MkdirAll
//   - Fail-fast validation on all operations
package io
