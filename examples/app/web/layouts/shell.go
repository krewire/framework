// Package layouts defines the application page shells (FRK-STR-006).
package layouts

import "github.com/krewire/framework/web/ssg"

const shellBody = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>{{.Title}}</title>
<link rel="stylesheet" href="/assets/ui.css">
<link rel="stylesheet" href="/assets/theme.css">
{{themeHead}}
</head>
<body>
<header class="topbar"><strong>Krewire App</strong><span class="spacer"></span>{{themeButton}}</header>
<main>{{.Content}}</main>
<footer class="footer">Served by a single krewire binary.</footer>
</body>
</html>`

// Shell is the default application layout: topbar with theme toggle, content,
// footer.
func Shell() ssg.Layout {
	return ssg.Layout{
		Name:  "Shell",
		Body:  shellBody,
		Style: `.topbar{display:flex;align-items:center;gap:1rem}.spacer{flex:1}`,
	}
}
