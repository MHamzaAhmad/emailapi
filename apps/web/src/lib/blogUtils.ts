/**
 * Blog utility functions for loading and parsing MDX blog posts
 */

export interface BlogPost {
    slug: string
    title: string
    description: string
    date: string
    tags: string[]
    readingTime: string
}

export interface BlogPostWithContent extends BlogPost {
    Content: React.ComponentType
}

// Type for the raw MDX module import
interface MDXModule {
    default: React.ComponentType
    frontmatter: {
        title: string
        description: string
        date: string
        tags?: string[]
        slug: string
    }
}

// Calculate reading time based on word count
function calculateReadingTime(content: string): string {
    const wordsPerMinute = 200
    const words = content.split(/\s+/).length
    const minutes = Math.ceil(words / wordsPerMinute)
    return `${minutes} min read`
}

// Get all blog posts metadata (sorted by date, newest first)
export async function getAllPosts(): Promise<BlogPost[]> {
    const modules = import.meta.glob<MDXModule>('../content/blog/*.mdx', { eager: true })

    const posts: BlogPost[] = Object.entries(modules).map(([path, module]) => {
        const slug = path.split('/').pop()?.replace('.mdx', '') || ''
        const { frontmatter } = module

        return {
            slug: frontmatter.slug || slug,
            title: frontmatter.title,
            description: frontmatter.description,
            date: frontmatter.date,
            tags: frontmatter.tags || [],
            readingTime: '5 min read', // Default, can be calculated from raw content if needed
        }
    })

    // Sort by date, newest first
    return posts.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
}

// Get a single blog post by slug
export async function getPostBySlug(slug: string): Promise<BlogPostWithContent | null> {
    const modules = import.meta.glob<MDXModule>('../content/blog/*.mdx', { eager: true })

    for (const [path, module] of Object.entries(modules)) {
        const fileSlug = path.split('/').pop()?.replace('.mdx', '') || ''
        const postSlug = module.frontmatter.slug || fileSlug

        if (postSlug === slug) {
            return {
                slug: postSlug,
                title: module.frontmatter.title,
                description: module.frontmatter.description,
                date: module.frontmatter.date,
                tags: module.frontmatter.tags || [],
                readingTime: '5 min read',
                Content: module.default,
            }
        }
    }

    return null
}

// Get related posts based on tags (excluding current post)
export async function getRelatedPosts(currentSlug: string, tags: string[], limit = 3): Promise<BlogPost[]> {
    const allPosts = await getAllPosts()

    return allPosts
        .filter(post => post.slug !== currentSlug)
        .filter(post => post.tags.some(tag => tags.includes(tag)))
        .slice(0, limit)
}

// Format date for display
export function formatDate(dateString: string): string {
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
    })
}
