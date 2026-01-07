import { createFileRoute } from '@tanstack/react-router'
import { Link } from '@tanstack/react-router'
import { getAllPosts, formatDate, type BlogPost } from '@/lib/blogUtils'
import { CalendarIcon, TagIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

export const Route = createFileRoute('/blog/')({
    loader: async () => {
        const posts = await getAllPosts()
        return { posts }
    },
    head: () => ({
        meta: [
            { title: 'Blog - SimpleEmailAPI | Email API Guides & Tutorials' },
            { name: 'description', content: 'Learn about email APIs, transactional emails, email delivery best practices, and more. Expert guides for developers building email infrastructure.' },
            { property: 'og:title', content: 'Blog - SimpleEmailAPI' },
            { property: 'og:description', content: 'Expert guides on email APIs, transactional emails, and email delivery for developers.' },
            { property: 'og:type', content: 'website' },
            { name: 'twitter:card', content: 'summary_large_image' },
            { name: 'twitter:title', content: 'Blog - SimpleEmailAPI' },
            { name: 'twitter:description', content: 'Expert guides on email APIs, transactional emails, and email delivery for developers.' },
        ],
    }),
    component: BlogIndexPage,
})

function BlogIndexPage() {
    const { posts } = Route.useLoaderData()

    return (
        <main className="mx-auto max-w-4xl px-6 py-16 sm:py-24">
            {/* Header */}
            <div className="mb-12">
                <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                    Blog
                </span>
                <h1 className="text-4xl sm:text-5xl font-bold tracking-tighter mb-4 text-foreground">
                    Insights & Guides
                </h1>
                <p className="text-muted-foreground text-base font-medium max-w-xl">
                    Learn about email APIs, transactional emails, and email delivery best practices.
                    Expert guides for developers building modern email infrastructure.
                </p>
            </div>

            {/* Posts Grid */}
            {posts.length === 0 ? (
                <div className="text-center py-20 border border-dashed border-border/60 rounded-lg bg-secondary/10">
                    <p className="text-muted-foreground text-sm">No blog posts yet. Check back soon!</p>
                </div>
            ) : (
                <div className="grid gap-6">
                    {posts.map((post) => (
                        <BlogPostCard key={post.slug} post={post} />
                    ))}
                </div>
            )}
        </main>
    )
}

function BlogPostCard({ post }: { post: BlogPost }) {
    return (
        <Link
            to="/blog/$slug"
            params={{ slug: post.slug }}
            className="group block p-6 rounded-lg border border-dashed border-border/60 bg-background hover:bg-secondary/20 hover:border-border transition-all"
        >
            <article>
                <div className="flex items-center gap-4 mb-3 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                        <HugeiconsIcon icon={CalendarIcon} size={12} />
                        {formatDate(post.date)}
                    </span>
                    <span className="text-border/60">•</span>
                    <span>{post.readingTime}</span>
                </div>

                <h2 className="text-xl font-semibold tracking-tight text-foreground group-hover:text-primary transition-colors mb-2">
                    {post.title}
                </h2>

                <p className="text-muted-foreground text-sm leading-relaxed mb-4 line-clamp-2">
                    {post.description}
                </p>

                {post.tags.length > 0 && (
                    <div className="flex items-center gap-2 flex-wrap">
                        <HugeiconsIcon icon={TagIcon} size={12} className="text-muted-foreground/60" />
                        {post.tags.slice(0, 3).map((tag) => (
                            <span
                                key={tag}
                                className="px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground bg-secondary/50 rounded-sm"
                            >
                                {tag}
                            </span>
                        ))}
                    </div>
                )}
            </article>
        </Link>
    )
}
