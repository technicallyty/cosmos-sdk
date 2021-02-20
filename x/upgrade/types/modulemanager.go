package types

// MigrationMap - maps module names to their consensus versions
type MigrationMap map[string]uint64

// ModuleManager - interface to assist with module upgrades
type ModuleManager interface {
	GetConsensusVersion() MigrationMap
}
