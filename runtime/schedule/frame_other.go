//go:build !js

package schedule

// DefaultFrame returns nil on non-js platforms: there is no animation frame
// to hook, so tests and servers drive Flush manually.
func DefaultFrame() func(func()) { return nil }
