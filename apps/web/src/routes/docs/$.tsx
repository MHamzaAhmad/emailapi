import { createFileRoute, notFound } from '@tanstack/react-router';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { source } from '@/lib/source';
import browserCollections from 'fumadocs-mdx:collections/browser';
import {
    DocsBody,
    DocsDescription,
    DocsPage,
    DocsTitle,
} from 'fumadocs-ui/layouts/docs/page';
import defaultMdxComponents from 'fumadocs-ui/mdx';
import { baseOptions } from '@/lib/layout.shared';

export const Route = createFileRoute('/docs/$')({
    component: Page,
    loader: async ({ params }) => {
        const slugs = params._splat?.split('/').filter(Boolean) ?? [];
        const page = source.getPage(slugs);
        if (!page) throw notFound();

        await clientLoader.preload(page.path);
        return { path: page.path };
    },
});

const clientLoader = browserCollections.docs.createClientLoader({
    component({ toc, frontmatter, default: MDX }) {
        return (
            <DocsPage toc={toc}>
                <DocsTitle>{frontmatter.title}</DocsTitle>
                <DocsDescription>{frontmatter.description}</DocsDescription>
                <DocsBody>
                    <MDX
                        components={{
                            ...defaultMdxComponents,
                        }}
                    />
                </DocsBody>
            </DocsPage>
        );
    },
});

function Page() {
    const data = Route.useLoaderData();
    const Content = clientLoader.getComponent(data.path);

    return (
        <DocsLayout {...baseOptions()} tree={source.pageTree}>
            <Content />
        </DocsLayout>
    );
}