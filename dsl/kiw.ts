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

export interface StyleBlock { lang: string; scoped: boolean; content: string }
export interface ScriptBlock { lang: string; hydrate: string; server: boolean; compute: boolean; content: string }

export interface KiwModule {
  frontmatter: Record<string, any>
  body: string
  styles: string[]
  scripts: string[]
  styleBlocks: StyleBlock[]
  scriptBlocks: ScriptBlock[]
  markdown: string[]
  raw: string
}

const styleRe = /<style([^>]*)>([\s\S]*?)<\/style>/gi
const scriptRe = /<script([^>]*)>([\s\S]*?)<\/script>/gi
const markdownRe = /<markdown[^>]*>([\s\S]*?)<\/markdown>/gi
const attrRe = /([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g

function parseAttrs(tag: string): Record<string, string> {
  const out: Record<string, string> = {}
  let m: RegExpExecArray | null
  attrRe.lastIndex = 0
  while ((m = attrRe.exec(tag)) !== null) {
    const key = m[1].toLowerCase()
    const val = m[2] ?? m[3] ?? m[4] ?? ""
    out[key] = val
  }
  return out
}

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
  const styleBlocks: StyleBlock[] = []
  body = body.replace(styleRe, (_m, attrs, inner) => {
    const a = parseAttrs(attrs)
    const content = inner.trim()
    styles.push(content)
    const lang = (a["lang"] || "css").toLowerCase()
    styleBlocks.push({ lang, scoped: "scoped" in a, content })
    return ""
  })

  const scripts: string[] = []
  const scriptBlocks: ScriptBlock[] = []
  body = body.replace(scriptRe, (_m, attrs, inner) => {
    const a = parseAttrs(attrs)
    const content = inner.trim()
    scripts.push(content)
    let lang = (a["lang"] || "js").toLowerCase()
    let hydrate = (a["hydrate"] || "").toLowerCase()
    const server = "server" in a
    const compute = "compute" in a
    if ((lang === "js" || lang === "ts" || lang === "go") && !server && !compute && !hydrate) hydrate = "load"
    scriptBlocks.push({ lang, hydrate, server, compute, content })
    return ""
  })

  const markdown: string[] = []
  body = body.replace(markdownRe, (_m, inner) => {
    markdown.push(inner.trim())
    // Keep markdown position by injecting raw markdown (Go side renders to HTML)
    // JS consumers can render with a markdown lib if needed.
    return "\n" + inner.trim() + "\n"
  })

  return {
    frontmatter,
    body: body.trim(),
    styles,
    scripts,
    styleBlocks,
    scriptBlocks,
    markdown,
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
