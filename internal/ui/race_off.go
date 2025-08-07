//go:build !race
// +build !race

package ui

// RaceEnabled is false when not built with -race
const RaceEnabled = false
