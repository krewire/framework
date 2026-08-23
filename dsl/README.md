# framework/dsl — Kiw DSL

`.kiw` DSL — file-based templating for every workload (SSG, email, config, etc.), not just `web/ssg`.

Import `github.com/krewire/framework/dsl`:

```go
import "github.com/krewire/framework/dsl"

mod, _ := dsl.ParseKiwFile("pages/index.kiw")
fmt.Println(mod.Frontmatter["title"]) // Landing
fmt.Println(mod.Body)    // <h1>{{.Title}}</h1>
fmt.Println(mod.Styles)  // ["h1{color:red}"]
```

JS/TS (mirrors Go, feels native):

```ts
import { parseKiw } from "./kiw.ts"
const mod = parseKiw(await Deno.readTextFile("pages/index.kiw"))
console.log(mod.frontmatter.title) // Landing
```

Format (Astro-like, Go+JS native):
- `---` YAML frontmatter (parseable by `gopkg.in/yaml.v3` in Go and `yaml` in JS)
- `html/template` body (`{{.Title}}`, `{{component "Badge"}}`)
- `<style>` scoped CSS, `<script>` client JS — extracted, not rendered inline

Used by `web/ssg` file pipeline (`LoadFromDir`), but also usable standalone for any template.
