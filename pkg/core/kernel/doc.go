// Package kernel implements Fluxor Runtime v2: an in-process kernel with typed components,
// unified AppContext, transport-agnostic Handler + Middleware, and optional plugins.
//
// Existing HTTP stack (GinRequestContext, BaseServer) remains for compatibility; new code can use
// Kernel + kernel.Handler and wire HTTP via web.GinKernelHandler(gocmd, handler).
//
// Event consumers: kernel.AdaptMessageHandler(kernel.Context(), h) or base context + message headers.
package kernel
