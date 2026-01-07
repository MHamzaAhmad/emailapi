/**
 * Custom MDX components for consistent blog styling
 */
import type { ComponentPropsWithoutRef } from 'react'

// Heading components with anchor links
function H1(props: ComponentPropsWithoutRef<'h1'>) {
    return (
        <h1
            className="text-3xl sm:text-4xl font-bold tracking-tighter mb-6 mt-12 first:mt-0 text-foreground"
            {...props}
        />
    )
}

function H2(props: ComponentPropsWithoutRef<'h2'>) {
    return (
        <h2
            className="text-2xl sm:text-3xl font-bold tracking-tight mb-4 mt-10 text-foreground border-b border-dashed border-border/40 pb-2"
            {...props}
        />
    )
}

function H3(props: ComponentPropsWithoutRef<'h3'>) {
    return (
        <h3
            className="text-xl font-semibold tracking-tight mb-3 mt-8 text-foreground"
            {...props}
        />
    )
}

function H4(props: ComponentPropsWithoutRef<'h4'>) {
    return (
        <h4
            className="text-lg font-semibold mb-2 mt-6 text-foreground"
            {...props}
        />
    )
}

// Paragraph
function P(props: ComponentPropsWithoutRef<'p'>) {
    return (
        <p
            className="text-muted-foreground leading-relaxed mb-4 text-base"
            {...props}
        />
    )
}

// Links
function A(props: ComponentPropsWithoutRef<'a'>) {
    const isExternal = props.href?.startsWith('http')
    return (
        <a
            className="text-primary hover:text-primary/80 underline underline-offset-4 decoration-primary/30 hover:decoration-primary/60 transition-colors"
            target={isExternal ? '_blank' : undefined}
            rel={isExternal ? 'noopener noreferrer' : undefined}
            {...props}
        />
    )
}

// Lists
function Ul(props: ComponentPropsWithoutRef<'ul'>) {
    return (
        <ul
            className="list-disc pl-6 mb-4 space-y-2 text-muted-foreground marker:text-muted-foreground/40"
            {...props}
        />
    )
}

function Ol(props: ComponentPropsWithoutRef<'ol'>) {
    return (
        <ol
            className="list-decimal pl-6 mb-4 space-y-2 text-muted-foreground marker:text-muted-foreground/60"
            {...props}
        />
    )
}

function Li(props: ComponentPropsWithoutRef<'li'>) {
    return (
        <li className="leading-relaxed" {...props} />
    )
}

// Code blocks
function Pre(props: ComponentPropsWithoutRef<'pre'>) {
    return (
        <pre
            className="bg-secondary/50 border border-dashed border-border/60 rounded-lg p-4 mb-4 overflow-x-auto text-sm font-mono"
            {...props}
        />
    )
}

function Code(props: ComponentPropsWithoutRef<'code'>) {
    // Inline code vs code block
    const isInline = typeof props.children === 'string'

    if (isInline) {
        return (
            <code
                className="bg-secondary/50 px-1.5 py-0.5 rounded text-sm font-mono text-foreground"
                {...props}
            />
        )
    }

    return <code className="text-foreground" {...props} />
}

// Blockquote
function Blockquote(props: ComponentPropsWithoutRef<'blockquote'>) {
    return (
        <blockquote
            className="border-l-4 border-primary/40 pl-4 py-2 my-6 italic text-muted-foreground bg-secondary/20 rounded-r-lg"
            {...props}
        />
    )
}

// Horizontal rule
function Hr(props: ComponentPropsWithoutRef<'hr'>) {
    return (
        <hr
            className="border-dashed border-border/60 my-8"
            {...props}
        />
    )
}

// Strong and emphasis
function Strong(props: ComponentPropsWithoutRef<'strong'>) {
    return (
        <strong className="font-semibold text-foreground" {...props} />
    )
}

function Em(props: ComponentPropsWithoutRef<'em'>) {
    return (
        <em className="italic" {...props} />
    )
}

// Image
function Img(props: ComponentPropsWithoutRef<'img'>) {
    return (
        <figure className="my-8">
            <img
                className="rounded-lg border border-dashed border-border/60 w-full"
                loading="lazy"
                {...props}
            />
            {props.alt && (
                <figcaption className="text-center text-sm text-muted-foreground mt-2">
                    {props.alt}
                </figcaption>
            )}
        </figure>
    )
}

// Table
function Table(props: ComponentPropsWithoutRef<'table'>) {
    return (
        <div className="overflow-x-auto my-6">
            <table
                className="w-full border-collapse text-sm"
                {...props}
            />
        </div>
    )
}

function Th(props: ComponentPropsWithoutRef<'th'>) {
    return (
        <th
            className="border border-dashed border-border/60 px-4 py-2 text-left font-semibold bg-secondary/30 text-foreground"
            {...props}
        />
    )
}

function Td(props: ComponentPropsWithoutRef<'td'>) {
    return (
        <td
            className="border border-dashed border-border/60 px-4 py-2 text-muted-foreground"
            {...props}
        />
    )
}

// Export all components for MDX usage
export const mdxComponents = {
    h1: H1,
    h2: H2,
    h3: H3,
    h4: H4,
    p: P,
    a: A,
    ul: Ul,
    ol: Ol,
    li: Li,
    pre: Pre,
    code: Code,
    blockquote: Blockquote,
    hr: Hr,
    strong: Strong,
    em: Em,
    img: Img,
    table: Table,
    th: Th,
    td: Td,
}
