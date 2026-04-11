//go:build ignore
// +build ignore

// Protobuf code generation script
// Run: go generate ./proto

package main

//go:generate protoc --go_out=. --go_opt=paths=source_relative envelope.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative auth.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative heartbeat.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative game_mahjong.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative game_service.proto
