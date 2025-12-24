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
import { useFumadocsLoader } from 'fumadocs-core/source/client';

export const Route = createFileRoute('/docs/$')({
    component: Page,
    loader: async ({ params }) => {
        const slugs = params._splat?.split('/') ?? [];

        // Get the page from the source
        const page = source.getPage(slugs);
        if (!page) throw notFound();

        // Serialize page tree and preload the client-side content
        const pageTree = await source.serializePageTree(source.getPageTree());
        await clientLoader.preload(page.path);

        return {
            path: page.path,
            pageTree,
        };
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
    const { pageTree } = useFumadocsLoader(data);
    const Content = clientLoader.getComponent(data.path);

    return (
        <DocsLayout {...baseOptions()} tree={pageTree}>
            <Content />
        </DocsLayout>
    );
}