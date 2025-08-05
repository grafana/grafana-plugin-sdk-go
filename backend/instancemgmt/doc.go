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
// The context-aware manager checks the "ttlInstanceManager" feature toggle
// from the Grafana configuration and automatically uses the appropriate
// underlying implementation. This allows runtime switching without requiring
// plugin restarts or static configuration.
package instancemgmt
