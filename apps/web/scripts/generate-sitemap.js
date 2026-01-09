import { readdirSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { join, parse } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));
const workspaceRoot = join(__dirname, '..');

function generateSitemap() {
    console.log('Generating sitemap...');

    // Generate blog routes
    const blogDir = join(workspaceRoot, 'src/content/blog');
    const blogRoutes = [];

    try {
        if (existsSync(blogDir)) {
            const files = readdirSync(blogDir);
            files.forEach((file) => {
                if (file.endsWith('.mdx')) {
                    const slug = parse(file).name;
                    blogRoutes.push(`/blog/${slug}`);
                }
            });
        } else {
            console.warn(`Blog directory not found: ${blogDir}`);
        }
    } catch (e) {
        console.warn('Could not read blog directory', e);
    }

    const staticRoutes = [
        '/',
        '/about',
        '/blog',
        '/privacy',
        '/terms',
    ];

    const allRoutes = [...staticRoutes, ...blogRoutes];
    const siteUrl = 'https://simpleemailapi.dev';

    const sitemapContent = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${allRoutes
            .map((route) => {
                return `  <url>
    <loc>${siteUrl}${route}</loc>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>
  </url>`;
            })
            .join('\n')}
</urlset>`;

    // Define output directories
    // We want to write to .output/public (for Vercel/Nitro) and maybe dist/client just in case
    const outDirs = [
        join(workspaceRoot, '.output/public'),
        join(workspaceRoot, 'dist/client')
    ];

    outDirs.forEach((dir) => {
        try {
            if (!existsSync(dir)) {
                // If directory doesn't exist, we can try creating it, but if .output/public is missing, build might have failed
                console.warn(`Target directory ${dir} does not exist. Creating it.`);
                mkdirSync(dir, { recursive: true });
            }

            writeFileSync(join(dir, 'sitemap.xml'), sitemapContent);
            console.log(`✓ Sitemap generated at ${dir}/sitemap.xml`);
        } catch (e) {
            console.warn(`Failed to write sitemap to ${dir}`, e);
        }
    });
}

function generateRobots() {
    console.log('Generating robots.txt...');

    // Default to production for build, unless strictly specified otherwise
    // But usually we build for production. 
    // If we want to support dev vs prod robots, we can check env.
    const isProduction = process.env.NODE_ENV === 'production' || process.argv.includes('--production');

    // Hardcoded content based on what we had
    const productionContent = `User-agent: *
Allow: /
Sitemap: https://simpleemailapi.dev/sitemap.xml
`;

    const developmentContent = `User-agent: *
Disallow: /
`;

    // Access process.env.VITE_USER_NODE_ENV or just NODE_ENV. 
    // In post-build script, we might assume production if running "vite build".
    // Let's assume production content for now as this is a build script.
    // Or we can check if it's a preview build. 
    // For Vercel deployments, it's production.
    const robotsContent = productionContent;

    const outDirs = [
        join(workspaceRoot, '.output/public'),
        join(workspaceRoot, 'dist/client')
    ];

    outDirs.forEach((dir) => {
        try {
            if (!existsSync(dir)) {
                mkdirSync(dir, { recursive: true });
            }

            writeFileSync(join(dir, 'robots.txt'), robotsContent);
            console.log(`✓ robots.txt generated at ${dir}/robots.txt`);
        } catch (e) {
            console.warn(`Failed to write robots.txt to ${dir}`, e);
        }
    });

}

generateSitemap();
generateRobots();
