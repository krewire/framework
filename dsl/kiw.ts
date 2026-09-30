/**
 * Kiw DSL — JS/TS parser (mirrors Go ssg/kiw.go)
 *
 * DSL:
 * ---
 * title: Landing
 * layout: Base
 * ---
 * <h1>{{title}}</h1>
 * <Navbar />
 * <style>h1{color:red}</style>
 * <script>console.log(1)</script>
 *
 * Frontmatter is YAML (parse with `yaml` npm package or `js-yaml`),
 * body is HTML with Go html/template `{{.Title}}` or JS `{title}`.
 * For JS, treat body as string template and replace {{.X}} / {{X}} / {X}.
 */

export interface StyleBlock { lang: string; scoped: boolean; content: string }
export interface ScriptBlock { lang: string; hydrate: string; server: boolean; compute: boolean; content: string }

export interface ComponentCall {
  name: string
  props?: Record<string, string>
  raw: string
}

export interface KiwModule {
  frontmatter: Record<string, any>
  body: string
  styles: string[]
  scripts: string[]
  styleBlocks: StyleBlock[]
  scriptBlocks: ScriptBlock[]
  markdown: string[]
  components?: ComponentCall[]
  componentNames?: string[]
  raw: string
}

const styleRe = /<style([^>]*)>([\s\S]*?)<\/style>/gi
const scriptRe = /<script([^>]*)>([\s\S]*?)<\/script>/gi
const markdownRe = /<markdown[^>]*>([\s\S]*?)<\/markdown>/gi
const attrRe = /([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g
const selfClosingCompRe = /<([A-Z][a-zA-Z0-9_]*)([^>]*?)\s*\/>/g
const pairedCompRe = /<([A-Z][a-zA-Z0-9_]*)([^>]*)>([\s\S]*?)<\/\1>/g
const mustacheCompRe = /\{\{\s*component\s+"([^"]+)"(?:\s+([^{}]+))?\s*\}\}/g
const compAttrRe = /([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|\{([^}]*)\}|([^\s"'=<>`]+)))?/g

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

function desugarComponent(name: string, attrStr: string, bodyContent: string, raw: string): [string, ComponentCall] {
  const call: ComponentCall = { name, props: {}, raw }
  const trimmedAttr = attrStr.trim()
  const trimmedBody = bodyContent.trim()

  if (!trimmedAttr && !trimmedBody) {
    return [`{{component "${name}"}}`, call]
  }

  if (trimmedAttr === "." && !trimmedBody) {
    call.props = { ".": "." }
    return [`{{component "${name}" .}}`, call]
  }

  if (trimmedAttr.startsWith("(dict") && !trimmedBody) {
    return [`{{component "${name}" ${trimmedAttr}}}`, call]
  }

  const dictPairs: string[] = []
  if (trimmedAttr) {
    compAttrRe.lastIndex = 0
    let sm: RegExpExecArray | null
    while ((sm = compAttrRe.exec(attrStr)) !== null) {
      if (!sm[1] || !sm[1].trim()) continue
      const key = sm[1]
      if (!sm[0].includes("=")) {
        call.props![key] = "true"
        dictPairs.push(`"${key}" true`)
        continue
      }
      let val = ""
      let isExpr = false
      if (sm[2] !== undefined) {
        val = sm[2]
        if (val.startsWith("{{") && val.endsWith("}}")) {
          val = val.slice(2, -2).trim()
          isExpr = true
        } else if (val.startsWith("{") && val.endsWith("}")) {
          val = val.slice(1, -1).trim()
          isExpr = true
        }
      } else if (sm[3] !== undefined) {
        val = sm[3]
      } else if (sm[4] !== undefined) {
        val = sm[4].trim()
        isExpr = true
      } else if (sm[5] !== undefined) {
        val = sm[5]
        if (val === "true" || val === "false" || !isNaN(Number(val)) || val.startsWith(".") || val.startsWith("$")) {
          isExpr = true
        }
      }

      call.props![key] = val
      if (isExpr || val === "true" || val === "false" || !isNaN(Number(val))) {
        dictPairs.push(`"${key}" ${val}`)
      } else {
        dictPairs.push(`"${key}" ${JSON.stringify(val)}`)
      }
    }
  }

  if (trimmedBody) {
    call.props!["Body"] = trimmedBody
    dictPairs.push(`"Body" ${JSON.stringify(trimmedBody)}`)
  }

  if (dictPairs.length === 0) {
    return [`{{component "${name}"}}`, call]
  }
  return [`{{component "${name}" (dict ${dictPairs.join(" ")})}}`, call]
}

export function desugarTemplate(src: string): string {
  let res = src.replace(selfClosingCompRe, (match, name, attrStr) => {
    return desugarComponent(name, attrStr || "", "", match)[0]
  })
  res = res.replace(pairedCompRe, (match, name, attrStr, body) => {
    return desugarComponent(name, attrStr || "", body || "", match)[0]
  })
  return res
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
    return "\n" + inner.trim() + "\n"
  })

  // Desugar <ComponentName ... /> to {{component "ComponentName" ...}}
  body = desugarTemplate(body)

  const components: ComponentCall[] = []
  const nameSet = new Set<string>()
  mustacheCompRe.lastIndex = 0
  let mm: RegExpExecArray | null
  while ((mm = mustacheCompRe.exec(body)) !== null) {
    const name = mm[1]
    components.push({ name, raw: mm[0] })
    nameSet.add(name)
  }

  return {
    frontmatter,
    body: body.trim(),
    styles,
    scripts,
    styleBlocks,
    scriptBlocks,
    markdown,
    components,
    componentNames: Array.from(nameSet).sort(),
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
