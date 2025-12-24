import { defineConfig } from 'vite'
import { devtools } from '@tanstack/devtools-vite'
import { tanstackStart } from '@tanstack/react-start/plugin/vite'
import viteReact from '@vitejs/plugin-react'
import viteTsConfigPaths from 'vite-tsconfig-paths'
import tailwindcss from '@tailwindcss/vite'
import mdx from 'fumadocs-mdx/vite';
import * as MdxConfig from './source.config';

const config = defineConfig({
  server: {
    port: 3000,
  },
  plugins: [
    mdx(MdxConfig),
    devtools(),
    // this is the plugin that enables path aliases
    viteTsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    tailwindcss(),
    // tanstackStart handles SSR/server integration - no need for separate nitro()
    tanstackStart(),
    // react's vite plugin must come after start's vite plugin
    viteReact(),
  ],
})

export default config

