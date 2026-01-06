import { defineConfig } from 'tsup'

export default defineConfig({
    entry: {
        index: 'src/index.ts',
        'worker-runtime': 'src/worker-runtime.ts',
    },
    format: ['cjs', 'esm'],
    dts: {
        entry: 'src/index.ts',
    },
    splitting: false, // Keep off - SDK consumers bundle this
    sourcemap: false, // Disable for production
    clean: true,
    minify: true, // Enable - reduces bundle size ~30-40%
    treeshake: true,
    external: ['worker_threads'], // Node.js built-in
    // Target modern runtimes
    target: 'node18',
})
