// Package extension defines public extension contracts and provider registries
// for Incident Investigator.
//
// # Extension model
//
// The runtime owns investigation state. Extensions register providers that
// supply capabilities—reasoners, archetypes, playbooks, reports, graph stores,
// and intelligence backends—without modifying core runtime code.
//
// # In-tree development
//
// Reference implementations live under internal/ and satisfy the documented
// contracts (for example internal/reasoning.Reasoner). Register providers via
// runtime.Option helpers such as WithReasonerRegistry.
//
// # Dependency direction
//
// This package must not import internal/. Application code inside this module
// wires internal implementations to runtime options at startup.
//
// See docs/extension-apis.md for lifecycle and ownership rules.
package extension
