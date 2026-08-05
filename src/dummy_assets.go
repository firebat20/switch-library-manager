package main

// Dummy definitions for astilectron-bundler generated symbols so go vet / go build can run during dev
var (
	Asset         func(name string) ([]byte, error)
	AssetDir      func(name string) ([]string, error)
	RestoreAssets func(dir, name string) error
)
