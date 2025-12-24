
import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

export function baseOptions(): BaseLayoutProps {
    return {
        nav: {
            title: 'EmailAPI Docs',
        },
        links: [
            {
                text: 'Documentation',
                url: '/docs',
                active: 'nested-url',
            },
        ],
    };
}
