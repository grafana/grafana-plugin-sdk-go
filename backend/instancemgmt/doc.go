// Package instancemgmt provides utilities for managing plugin instances.
//
// This package offers several instance manager implementations:
//
//  1. Standard Instance Manager (New): Uses sync.Map for caching instances
//     and disposes them when they need updates.
//
//  2. TTL Instance Manager (NewTTLInstanceManager): Uses TTL-based caching
//     that automatically evicts instances after a configurable time period.
//
//  3. Instance Manager Wrapper (NewInstanceManagerWrapper):
//     Dynamically selects between standard and TTL managers based on
//     feature toggles from the Grafana config in the context.
//
// The context-aware manager checks the "ttlPluginInstanceManager" feature toggle
// from the Grafana configuration and automatically uses the appropriate
// underlying implementation. This allows runtime switching without requiring
// plugin restarts or static configuration.
// Use Acquire and defer its release function, or use Do, to keep an instance
// alive for an entire operation. The SDK's automatic handler manager does this
// for each handler call. Get remains available for compatibility, but cannot
// track how long the returned instance is used. Context cancellation alone does
// not release an acquired instance; the handler must actually finish.
//
// Note: Both the standard and TTL instance managers maintain separate caches.
// If the feature toggle for TTL instance manager is changed at runtime,
// instances are not migrated between managers. This can result in duplicate
// instances for the same plugin context until the old ones are disposed or evicted.
package instancemgmt
