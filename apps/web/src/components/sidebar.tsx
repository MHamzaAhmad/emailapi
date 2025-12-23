import { Link, useRouterState } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Home01Icon,
    Key01Icon,
    GlobeIcon,
    WebhookIcon,
    Book02Icon,
    Settings01Icon,
    ArrowRight01Icon,
} from '@hugeicons/core-free-icons'
import type { IconSvgElement } from '@hugeicons/react'
import { cn } from '@/lib/utils'
import { Separator } from '@/components/ui/separator'

interface NavItem {
    title: string
    href: string
    icon: IconSvgElement
    external?: boolean
}

const mainNav: NavItem[] = [
    {
        title: 'Dashboard',
        href: '/dashboard',
        icon: Home01Icon,
    },
    {
        title: 'Domains',
        href: '/domains',
        icon: GlobeIcon,
    },
    {
        title: 'API Keys',
        href: '/api-keys',
        icon: Key01Icon,
    },
    {
        title: 'Webhooks',
        href: '/webhooks',
        icon: WebhookIcon,
    },
]

const secondaryNav: NavItem[] = [
    {
        title: 'Documentation',
        href: '/docs',
        icon: Book02Icon,
    },
    {
        title: 'Settings',
        href: '/settings',
        icon: Settings01Icon,
    },
]

export function Sidebar() {
    const routerState = useRouterState()
    const currentPath = routerState.location.pathname

    return (
        <aside className="fixed left-0 top-0 z-40 h-screen w-[200px] border-r border-border bg-sidebar">
            <div className="flex h-full flex-col">
                {/* Logo */}
                <div className="flex h-12 items-center border-b border-border px-4">
                    <Link to="/" className="flex items-center gap-2">
                        <div className="flex items-center gap-1.5">
                            <div className="h-5 w-5 rounded border border-foreground/30 flex items-center justify-center">
                                <HugeiconsIcon icon={ArrowRight01Icon} size={12} strokeWidth={1.5} />
                            </div>
                            <span className="text-sm font-medium tracking-tight">emailapi</span>
                        </div>
                    </Link>
                </div>

                {/* Main Navigation */}
                <nav className="flex-1 space-y-1 p-2">
                    <div className="space-y-0.5">
                        {mainNav.map((item) => (
                            <Link
                                key={item.href}
                                to={item.href}
                                className={cn(
                                    'nav-item',
                                    currentPath === item.href && 'nav-item-active'
                                )}
                            >
                                <HugeiconsIcon icon={item.icon} size={14} strokeWidth={1.5} />
                                <span>{item.title}</span>
                            </Link>
                        ))}
                    </div>

                    <Separator className="my-3" />

                    <div className="space-y-0.5">
                        {secondaryNav.map((item) => (
                            <Link
                                key={item.href}
                                to={item.href}
                                className={cn(
                                    'nav-item',
                                    currentPath.startsWith(item.href) && 'nav-item-active'
                                )}
                            >
                                <HugeiconsIcon icon={item.icon} size={14} strokeWidth={1.5} />
                                <span>{item.title}</span>
                            </Link>
                        ))}
                    </div>
                </nav>

                {/* Footer */}
                <div className="border-t border-border p-2">
                    <div className="px-2 py-1.5">
                        <p className="text-2xs text-muted-foreground">v1.0.0-dev</p>
                    </div>
                </div>
            </div>
        </aside>
    )
}
