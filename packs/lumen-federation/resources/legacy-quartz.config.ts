import { QuartzConfig } from "./quartz/cfg"
import * as Plugin from "./quartz/plugins"
import { QuartzFilterPlugin } from "./quartz/plugins/types"

const publicWikiPaths = new Set([
  "index.md",
  "cities/aurel.md",
  "cities/cindral.md",
  "cities/edenfall.md",
  "cities/iustar.md",
  "cities/lucera.md",
  "cities/noxmere.md",
  "cities/veyr.md",
  "glossary/firewood.md",
  "glossary/monsters.md",
  "glossary/slang.md",
  "systems/civilization-resonance-pressure.md",
  "systems/city-infrastructure-matrix.md",
  "systems/dimming-choice-consequences.md",
  "systems/dimming-pressure-calendar.md",
  "systems/intercity-dependency-network.md",
  "systems/lex-humanitas.md",
  "systems/light-zone-class-regime.md",
  "systems/marrow-afterglow-fuel-cycle.md",
  "systems/the-great-dimming.md",
  "worlds/geopolitical-structure.md",
  "worlds/lumen-federation.md",
])

const PublicWikiOnly: QuartzFilterPlugin = () => ({
  name: "PublicWikiOnly",
  shouldPublish(_ctx, [_tree, vfile]) {
    return publicWikiPaths.has(vfile.data.relativePath)
  },
})

const config: QuartzConfig = {
  configuration: {
    pageTitle: "World Lore",
    pageTitleSuffix: "",
    enableSPA: true,
    enablePopovers: true,
    analytics: null,
    locale: "ko-KR",
    baseUrl: process.env.QUARTZ_BASE_URL ?? "localhost:8088",
    ignorePatterns: ["private", "templates", ".obsidian", "raw", "drafts"],
    defaultDateType: "modified",
    theme: {
      fontOrigin: "googleFonts",
      cdnCaching: true,
      typography: {
        header: "Schibsted Grotesk",
        body: "Source Sans Pro",
        code: "IBM Plex Mono",
      },
      colors: {
        lightMode: {
          light: "#faf8f8",
          lightgray: "#e5e5e5",
          gray: "#b8b8b8",
          darkgray: "#4e4e4e",
          dark: "#2b2b2b",
          secondary: "#284b63",
          tertiary: "#84a59d",
          highlight: "rgba(143, 159, 169, 0.15)",
          textHighlight: "#fff23688",
        },
        darkMode: {
          light: "#161618",
          lightgray: "#393639",
          gray: "#646464",
          darkgray: "#d4d4d4",
          dark: "#ebebec",
          secondary: "#7b97aa",
          tertiary: "#84a59d",
          highlight: "rgba(143, 159, 169, 0.15)",
          textHighlight: "#b3aa0288",
        },
      },
    },
  },
  plugins: {
    transformers: [
      Plugin.FrontMatter(),
      Plugin.CreatedModifiedDate({
        priority: ["frontmatter"],
      }),
      Plugin.SyntaxHighlighting({
        theme: {
          light: "github-light",
          dark: "github-dark",
        },
        keepBackground: false,
      }),
      Plugin.ObsidianFlavoredMarkdown({ enableInHtmlEmbed: false }),
      Plugin.GitHubFlavoredMarkdown(),
      Plugin.TableOfContents(),
      Plugin.CrawlLinks({ markdownLinkResolution: "shortest" }),
      Plugin.Description(),
      Plugin.Latex({ renderEngine: "katex" }),
    ],
    filters: [Plugin.RemoveDrafts(), PublicWikiOnly()],
    emitters: [
      Plugin.AliasRedirects(),
      Plugin.ComponentResources(),
      Plugin.ContentPage(),
      Plugin.FolderPage(),
      Plugin.TagPage(),
      Plugin.ContentIndex({
        enableSiteMap: true,
        enableRSS: true,
      }),
      Plugin.Assets(),
      Plugin.Static(),
      Plugin.Favicon(),
      Plugin.NotFoundPage(),
    ],
  },
}

export default config
