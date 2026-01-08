import { defineConfig } from 'vite'
import { robots } from 'vite-plugin-robots'
import { devtools } from '@tanstack/devtools-vite'
import { tanstackStart } from '@tanstack/react-start/plugin/vite'
import viteReact from '@vitejs/plugin-react'
import viteTsConfigPaths from 'vite-tsconfig-paths'
import tailwindcss from '@tailwindcss/vite'
import { nitro } from 'nitro/vite'
import mdx from '@mdx-js/rollup'
import remarkFrontmatter from 'remark-frontmatter'
import remarkMdxFrontmatter from 'remark-mdx-frontmatter'
import remarkGfm from 'remark-gfm'
import { readdirSync, writeFileSync, mkdirSync } from 'node:fs'
import { join, parse } from 'node:path'

// Generate blog routes
const blogDir = join(process.cwd(), 'src/content/blog')
const blogRoutes: string[] = []

try {
  const files = readdirSync(blogDir)
  files.forEach((file) => {
    if (file.endsWith('.mdx')) {
      const slug = parse(file).name
      blogRoutes.push(`/blog/${slug}`)
    }
  })
} catch (e) {
  console.warn('Could not read blog directory', e)
}

const staticRoutes = [
  '/',
  '/about',
  '/blog',
  '/privacy',
  '/terms',
]

const allRoutes = [...staticRoutes, ...blogRoutes]
const siteUrl = 'https://simpleemailapi.dev'

const customSitemapPlugin = () => {
  return {
    name: 'custom-sitemap-plugin',
    closeBundle: () => {
      const sitemapContent = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${allRoutes
          .map((route) => {
            return `  <url>
    <loc>${siteUrl}${route}</loc>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>
  </url>`
          })
          .join('\n')}
</urlset>`

      const outDirs = ['.output/public', 'dist/client']

      outDirs.forEach(dir => {
        try {
          const fullPath = join(process.cwd(), dir)
          mkdirSync(fullPath, { recursive: true })
          writeFileSync(join(fullPath, 'sitemap.xml'), sitemapContent)
          console.log(`✓ Sitemap generated at ${dir}/sitemap.xml`)
        } catch (e) {
          console.warn(`Failed to write sitemap to ${dir}`, e)
        }
      })
    },
  }
}

const config = defineConfig({
  server: {
    port: 3000,
  },
  define: {
    'process.env': {},
  },
  plugins: [
    customSitemapPlugin(),
    robots({}),
    devtools(),
    // this is the plugin that enables path aliases
    viteTsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    tailwindcss(),
    tanstackStart(),
    nitro(),
    // MDX plugin for blog content - must come before React plugin
    mdx({
      remarkPlugins: [remarkFrontmatter, remarkMdxFrontmatter, remarkGfm],
    }),
    // react's vite plugin must come after start's vite plugin
    viteReact(),
  ],
})


export default config

