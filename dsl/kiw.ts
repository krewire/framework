/**
 * Kiw DSL — JS/TS parser (mirrors Go ssg/kiw.go)
 *
 * DSL:
 * ---
 * title: Landing
 * layout: Base
 * ---
 * <h1>{{title}}</h1>
 * <style>h1{color:red}</style>
 * <script>console.log(1)</script>
 *
 * Frontmatter is YAML (parse with `yaml` npm package or `js-yaml`),
 * body is HTML with Go html/template `{{.Title}}` or JS `{title}`.
 * For JS, treat body as string template and replace {{.X}} / {{X}} / {X}.
 */

export interface KiwModule {
  frontmatter: Record<string, any>
  body: string
  styles: string[]
  scripts: string[]
  raw: string
}

const styleRe = /<style[^>]*>([\s\S]*?)<\/style>/gi
const scriptRe = /<script[^>]*>([\s\S]*?)<\/script>/gi

export function parseKiw(src: string): KiwModule {
  let frontmatter: Record<string, any> = {}
  let body = src
  const trimmed = src.trimStart()
  if (trimmed.startsWith("---")) {
    const rest = trimmed.slice(3)
    const idx = rest.indexOf("\n---")
    if (idx >= 0) {
      const fmRaw = rest.slice(0, idx)
      body = rest.slice(idx + 4).replace(/^\r?\n/, "")
      try {
        // minimal YAML parse without deps for MVP: key: value lines
        frontmatter = parseYamlMinimal(fmRaw)
      } catch {}
    }
  }

  const styles: string[] = []
  body = body.replace(styleRe, (_m, inner) => {
    styles.push(inner.trim())
    return ""
  })

  const scripts: string[] = []
  body = body.replace(scriptRe, (_m, inner) => {
    scripts.push(inner.trim())
    return ""
  })

  return {
    frontmatter,
    body: body.trim(),
    styles,
    scripts,
    raw: src,
  }
}

function parseYamlMinimal(src: string): Record<string, any> {
  const out: Record<string, any> = {}
  for (const line of src.split("\n")) {
    const t = line.trim()
    if (!t || t.startsWith("#")) continue
    const colon = t.indexOf(":")
    if (colon === -1) continue
    const k = t.slice(0, colon).trim()
    const v = t.slice(colon + 1).trim().replace(/^["']|["']$/g, "")
    out[k] = v
  }
  return out
}

export function parseKiwFile(src: string): KiwModule {
  return parseKiw(src)
}

// Example (native JS):
// import { parseKiw } from "./kiw.ts"
// const mod = parseKiw(await Deno.readTextFile("pages/index.kiw"))
// console.log(mod.frontmatter.title) // "Landing"
// document.body.innerHTML = mod.body.replace("{{.Title}}", mod.frontmatter.title)
