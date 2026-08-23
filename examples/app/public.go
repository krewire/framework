// Package public embeds the application's static assets served as-is
// (FRK-STR-007). The seam lives at the module root because //go:embed cannot
// cross parent directories, and internal/app imports it by path.
package public

import "embed"

//go:embed all:public
var Files embed.FS
