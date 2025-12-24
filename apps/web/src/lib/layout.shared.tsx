import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { PackageIcon } from '@hugeicons/core-free-icons';
import { HugeiconsIcon } from '@hugeicons/react';

export function baseOptions(): BaseLayoutProps {
    return {
        nav: {
            title: (
                <div className="flex items-center gap-2 font-bold tracking-tight">
                    <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                        <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
                    </div>
                    <span className="text-sm">emailapi</span>
                </div>
            ),
            url: '/',
        },
        links: [
            {
                text: 'API Reference',
                url: '/docs/api-reference',
            },
            {
                text: 'Webhooks',
                url: '/webhooks',
            },
        ],
    };
}

