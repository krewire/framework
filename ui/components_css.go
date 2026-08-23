package ui

// Shared design tokens used by all component styles. Consumers include these
// alongside the theme palette so the components resolve their variables.
const uiLayoutCSS = `:root {
  --ui-radius: 14px;
  --ui-radius-sm: 8px;
  --ui-radius-round: 999px;
  --ui-transition: 0.22s ease;
  --ui-max-width: 1200px;
  --ui-gap: 1.5rem;
  --ui-font-sans: 'Inter', system-ui, -apple-system, sans-serif;
  --ui-font-display: 'Outfit', 'Inter', system-ui, sans-serif;
  --ui-font-mono: 'Fira Code', ui-monospace, monospace;
}
* { box-sizing: border-box; }
body { margin: 0; font-family: var(--ui-font-sans); }
a { text-decoration: none; color: inherit; }`

// ContainerCSS styles the max-width wrapper.
const ContainerCSS = `.container {
  width: 100%;
  max-width: var(--ui-max-width);
  margin: 0 auto;
  padding: 0 2rem;
}
@media (max-width: 640px) {
  .container { padding: 0 1.25rem; }
}`

// GridCSS styles responsive grids by column count.
const GridCSS = `.grid { display: grid; gap: var(--ui-gap); }
.grid-1 { grid-template-columns: repeat(1, 1fr); }
.grid-2 { grid-template-columns: repeat(2, 1fr); }
.grid-3 { grid-template-columns: repeat(3, 1fr); }
.grid-4 { grid-template-columns: repeat(4, 1fr); }
.grid-5 { grid-template-columns: repeat(5, 1fr); }
.grid-6 { grid-template-columns: repeat(6, 1fr); }
@media (max-width: 900px) {
  .grid { grid-template-columns: repeat(2, 1fr) !important; }
}
@media (max-width: 560px) {
  .grid { grid-template-columns: 1fr !important; }
}`

// HeaderCSS styles the top navigation bar.
const HeaderCSS = `.ui-header {
  position: sticky;
  top: 0;
  z-index: 100;
  border-bottom: 1px solid var(--base-3);
  background-color: var(--base-1);
}
.header-inner {
  max-width: var(--ui-max-width);
  margin: 0 auto;
  padding: 0 2rem;
  min-height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.25rem;
}
.header-brand { display: flex; align-items: center; gap: 0.6rem; }
.header-nav { display: flex; align-items: center; }
.header-actions { display: flex; align-items: center; gap: 0.75rem; }
.nav-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  gap: 1.25rem;
  align-items: center;
}
.nav-link {
  font-weight: 500;
  font-size: 0.9rem;
  color: var(--base-2-content);
  transition: color var(--ui-transition);
}
.nav-link:hover { color: var(--primary); }
.nav-item.is-active .nav-link { color: var(--primary); }
@media (max-width: 640px) {
  .header-inner { padding: 0 1.25rem; }
  .nav-list { gap: 0.9rem; }
}`

// FooterCSS styles the footer bar.
const FooterCSS = `.ui-footer {
  border-top: 1px solid var(--base-3);
  background-color: var(--base-2);
  padding: 2.5rem 0;
}
.footer-inner {
  max-width: var(--ui-max-width);
  margin: 0 auto;
  padding: 0 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1.5rem;
  color: var(--base-2-content);
  font-size: 0.82rem;
}
.footer-left, .footer-center, .footer-right { display: flex; align-items: center; gap: 0.5rem; }
@media (max-width: 640px) {
  .footer-inner { flex-direction: column; text-align: center; padding: 0 1.25rem; }
}`

// CardCSS styles bordered cards.
const CardCSS = `.card {
  background-color: var(--base-1);
  border: 1px solid var(--base-3);
  border-radius: var(--ui-radius);
  overflow: hidden;
  transition: all var(--ui-transition);
}
.card:hover { transform: translateY(-4px); border-color: var(--primary); box-shadow: 0 8px 24px rgba(0, 200, 83, 0.07); }
.card-header, .card-body, .card-footer { padding: 1.5rem; }
.card-header { border-bottom: 1px solid var(--base-3); font-weight: 600; }
.card-footer { border-top: 1px solid var(--base-3); }`

// SectionCSS styles vertical rhythm sections.
const SectionCSS = `.section { padding: 5rem 2rem; }`

// HeroCSS styles the prominent hero section.
const HeroCSS = `.hero {
  position: relative;
  padding: 7rem 2rem 5rem;
  overflow: hidden;
  background-color: var(--base-1);
}
.hero-inner {
  max-width: var(--ui-max-width);
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4rem;
  align-items: center;
}
.hero-badge { margin-bottom: 1.5rem; }
.hero-title {
  font-family: var(--ui-font-display);
  font-size: clamp(2.4rem, 5vw, 3.6rem);
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -0.04em;
  margin: 0 0 1.25rem;
  color: var(--base-1-content);
}
.hero-subtitle {
  font-size: 1.1rem;
  line-height: 1.65;
  color: var(--base-2-content);
  margin: 0 0 2.25rem;
  max-width: 480px;
}
.hero-cta { display: flex; gap: 1rem; flex-wrap: wrap; }
.hero-visual { position: relative; }
@media (max-width: 900px) {
  .hero { padding: 5rem 2rem 4rem; }
  .hero-inner { grid-template-columns: 1fr; gap: 2.5rem; }
  .hero-subtitle { max-width: 100%; }
}`

// BadgeCSS styles inline labels.
const BadgeCSS = `.badge {
  display: inline-block;
  padding: 0.3rem 0.85rem;
  border-radius: var(--ui-radius-round);
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.badge-primary { color: var(--primary); background: rgba(0, 200, 83, 0.1); border: 1px solid rgba(0, 200, 83, 0.2); }
.badge-neutral { color: var(--neutral-content); background: var(--neutral); border: 1px solid var(--base-3); }
.badge-success { color: var(--success); background: rgba(21, 128, 61, 0.1); border: 1px solid rgba(21, 128, 61, 0.2); }
.badge-info { color: var(--info); background: rgba(13, 148, 136, 0.1); border: 1px solid rgba(13, 148, 136, 0.2); }
.badge-warning { color: var(--warning); background: rgba(245, 158, 11, 0.1); border: 1px solid rgba(245, 158, 11, 0.2); }
.badge-error { color: var(--error); background: rgba(185, 28, 28, 0.1); border: 1px solid rgba(185, 28, 28, 0.2); }`

// ButtonCSS styles buttons and links.
const ButtonCSS = `.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.75rem 1.6rem;
  font-family: inherit;
  font-weight: 600;
  font-size: 0.95rem;
  border-radius: var(--ui-radius-sm);
  text-decoration: none;
  border: none;
  cursor: pointer;
  transition: all var(--ui-transition);
}
.btn-primary { background: var(--primary); color: var(--primary-content); box-shadow: 0 4px 16px rgba(92, 255, 142, 0.25); }
.btn-primary:hover { transform: translateY(-2px); box-shadow: 0 6px 22px rgba(92, 255, 142, 0.4); }
.btn-ghost { background: var(--base-2); color: var(--base-1-content); border: 1px solid var(--base-3); }
.btn-ghost:hover { background: var(--base-3); border-color: var(--primary); transform: translateY(-2px); }
.btn-outline { background: transparent; color: var(--primary); border: 1px solid var(--primary); }
.btn-outline:hover { background: var(--primary); color: var(--primary-content); }`

// ThemeToggleWrapCSS styles the theme-toggle wrapper container.
const ThemeToggleWrapCSS = `.theme-toggle-wrap { display: inline-flex; align-items: center; }`

// ComponentsCSS returns the shared layout primitives plus every component
// style — including the theme mode variables and toggle button — ready for
// inclusion in a page's global stylesheet.
func ComponentsCSS() string {
	return uiLayoutCSS + "\n" +
		ThemeModeVarsCSS + "\n" +
		ThemeToggleCSS + "\n" +
		ContainerCSS + "\n" +
		GridCSS + "\n" +
		HeaderCSS + "\n" +
		FooterCSS + "\n" +
		CardCSS + "\n" +
		SectionCSS + "\n" +
		HeroCSS + "\n" +
		BadgeCSS + "\n" +
		ButtonCSS + "\n" +
		ThemeToggleWrapCSS
}
