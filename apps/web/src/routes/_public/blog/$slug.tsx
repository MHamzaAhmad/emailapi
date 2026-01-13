import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import { getPostBySlug, getRelatedPosts, formatDate, type BlogPost } from '@/lib/blogUtils'
import { mdxComponents } from '@/components/mdx-components'
import { CalendarIcon, ArrowLeftIcon, TagIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

export const Route = createFileRoute('/_public/blog/$slug')({
    loader: async ({ params }: { params: { slug: string } }) => {
        const post = await getPostBySlug(params.slug)
        if (!post) {
            throw notFound()
        }
        const relatedPosts = await getRelatedPosts(post.slug, post.tags, 3)
        return { post, relatedPosts } as any
    },
    head: ({ loaderData }) => {
        if (!loaderData?.post) {
            return { meta: [{ title: 'Post Not Found - SimpleEmailAPI Blog' }] }
        }
        const { post } = loaderData
        return {
            meta: [
                { title: `${post.title} - SimpleEmailAPI Blog` },
                { name: 'description', content: post.description },
                { name: 'keywords', content: post.tags.join(', ') },
                { property: 'og:title', content: post.title },
                { property: 'og:description', content: post.description },
                { property: 'og:type', content: 'article' },
                { property: 'article:published_time', content: post.date },
                { property: 'article:tag', content: post.tags.join(', ') },
                { name: 'twitter:card', content: 'summary_large_image' },
                { name: 'twitter:title', content: post.title },
                { name: 'twitter:description', content: post.description },
            ],
        }
    },
    component: BlogPostPage,
    notFoundComponent: PostNotFound,
})

function BlogPostPage() {
    const { post, relatedPosts } = Route.useLoaderData()
    // @ts-ignore - Content is a component
    const Content = post.Content

    return (
        <main className="mx-auto max-w-3xl px-6 py-16 sm:py-24">
            {/* Breadcrumb */}
            <Link
                to="/blog"
                className="inline-flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors mb-8"
            >
                <HugeiconsIcon icon={ArrowLeftIcon} size={14} />
                Back to Blog
            </Link>

            {/* Article Header */}
            <header className="mb-12">
                <div className="flex items-center gap-4 mb-4 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                        <HugeiconsIcon icon={CalendarIcon} size={12} />
                        {formatDate(post.date)}
                    </span>
                    <span className="text-border/60">•</span>
                    <span>{post.readingTime}</span>
                </div>

                <h1 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-4 text-foreground">
                    {post.title}
                </h1>

                <p className="text-muted-foreground text-base leading-relaxed mb-6">
                    {post.description}
                </p>

                {post.tags.length > 0 && (
                    <div className="flex items-center gap-2 flex-wrap">
                        <HugeiconsIcon icon={TagIcon} size={12} className="text-muted-foreground/60" />
                        {post.tags.map((tag: string) => (
                            <span
                                key={tag}
                                className="px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground bg-secondary/50 rounded-sm"
                            >
                                {tag}
                            </span>
                        ))}
                    </div>
                )}
            </header>

            {/* Article Content */}
            <article className="prose-custom">
                {/* @ts-ignore - MDX component types */}
                <Content components={mdxComponents} />
            </article>

            {/* Related Posts */}
            {relatedPosts.length > 0 && (
                <section className="mt-16 pt-12 border-t border-dashed border-border/40">
                    <h2 className="text-xl font-semibold tracking-tight mb-6 text-foreground">
                        Related Articles
                    </h2>
                    <div className="grid gap-4">
                        {relatedPosts.map((relatedPost: BlogPost) => (
                            <RelatedPostCard key={relatedPost.slug} post={relatedPost} />
                        ))}
                    </div>
                </section>
            )}

            {/* CTA Section */}
            <section className="mt-16 p-8 rounded-lg border border-dashed border-border/60 bg-secondary/10 text-center">
                <h3 className="text-xl font-semibold tracking-tight mb-3 text-foreground">
                    Ready to simplify your email infrastructure?
                </h3>
                <p className="text-muted-foreground text-sm mb-6 max-w-md mx-auto">
                    Join thousands of developers using SimpleEmailAPI for transactional and bulk email delivery.
                </p>
                <Link
                    to="/"
                    className="inline-flex h-9 px-6 items-center justify-center rounded-md bg-primary text-primary-foreground text-xs font-semibold shadow-sm hover:bg-primary/90 transition-colors"
                >
                    Get Started Free
                </Link>
            </section>
        </main>
    )
}

function RelatedPostCard({ post }: { post: BlogPost }) {
    return (
        <Link
            to="/blog/$slug"
            params={{ slug: post.slug }}
            className="group flex items-start gap-4 p-4 rounded-md border border-transparent hover:border-border/40 hover:bg-secondary/20 transition-all"
        >
            <div className="flex-1">
                <h3 className="text-sm font-medium text-foreground group-hover:text-primary transition-colors mb-1">
                    {post.title}
                </h3>
                <p className="text-xs text-muted-foreground line-clamp-1">
                    {post.description}
                </p>
            </div>
            <span className="text-xs text-muted-foreground shrink-0">
                {formatDate(post.date)}
            </span>
        </Link>
    )
}

function PostNotFound() {
    return (
        <main className="mx-auto max-w-3xl px-6 py-24 text-center">
            <h1 className="text-3xl font-bold tracking-tighter mb-4 text-foreground">
                Post Not Found
            </h1>
            <p className="text-muted-foreground mb-8">
                The blog post you're looking for doesn't exist or has been moved.
            </p>
            <Link
                to="/blog"
                className="inline-flex h-9 px-6 items-center justify-center rounded-md bg-primary text-primary-foreground text-xs font-semibold shadow-sm hover:bg-primary/90 transition-colors"
            >
                Back to Blog
            </Link>
        </main>
    )
}
