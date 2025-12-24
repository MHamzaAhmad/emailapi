import { Link, useRouterState } from '@tanstack/react-router'
import { UserButton } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    PackageIcon,
    Home01Icon,
    GlobeIcon,
    Key01Icon,
    WebhookIcon,
    Book02Icon,
    Settings01Icon
} from '@hugeicons/core-free-icons'
import { Button } from '@/components/ui/button'
import { ModeToggle } from '@/components/mode-toggle'

interface ShellProps {
    children: React.ReactNode
}

const mainNav = [
    { title: 'Dashboard', href: '/dashboard', icon: Home01Icon },
    { title: 'Domains', href: '/domains', icon: GlobeIcon },
    { title: 'API Keys', href: '/api-keys', icon: Key01Icon },
    { title: 'Webhooks', href: '/webhooks', icon: WebhookIcon },
]

export function Shell({ children }: ShellProps) {
    const routerState = useRouterState()
    const currentPath = routerState.location.pathname

    return (
        <div className="min-h-screen bg-background flex flex-col font-sans">
            {/* Top Navigation Bar - Ultra Minimal */}
            <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
                <div className="mx-auto flex h-14 max-w-7xl items-center px-4 md:px-6 gap-4">
                    <Link to="/" className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:text-foreground transition-colors mr-6">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                            <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
                        </div>
                        <span className="hidden md:inline-block text-sm">emailapi</span>
                    </Link>

                    <nav className="flex items-center gap-1">
                        {mainNav.map((item) => {
                            const isActive = currentPath === item.href || (item.href !== '/dashboard' && currentPath.startsWith(item.href))
                            return (
                                <Link
                                    key={item.href}
                                    to={item.href}
                                    className={`
                                        flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-200
                                        ${isActive
                                            ? 'bg-secondary text-foreground shadow-sm ring-1 ring-black/5 dark:ring-white/10'
                                            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                                        }
                                    `}
                                >
                                    <span>{item.title}</span>
                                </Link>
                            )
                        })}
                    </nav>

                    <div className="flex-1" />

                    <div className="flex items-center gap-3">
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground" asChild>
                            {/* @ts-expect-error docs route */}
                            <Link to="/docs">
                                <HugeiconsIcon icon={Book02Icon} size={16} />
                            </Link>
                        </Button>
                        <ModeToggle />
                        <div className="h-4 w-[1px] bg-border/60 mx-1" />
                        <UserButton
                            appearance={{
                                elements: {
                                    avatarBox: "h-7 w-7 rounded-full ring-2 ring-background hover:ring-muted transition-all"
                                }
                            }}
                        />
                    </div>
                </div>
            </header>

            {/* Breadcrumbs & Content */}
            <main className="flex-1">
                <div className="mx-auto max-w-7xl px-4 md:px-6 py-6 font-sans">

                    <div className="animate-in fade-in slide-in-from-bottom-2 duration-700 ease-out fill-mode-backwards">
                        {children}
                    </div>
                </div>
            </main>
        </div>
    )
}


interface PageHeaderProps {
    title: string
    description?: string
    actions?: React.ReactNode
}

export function PageHeader({ title, description, actions }: PageHeaderProps) {
    return (
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between mb-8">
            <div className="space-y-1">
                <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
                {description && (
                    <p className="text-base text-muted-foreground max-w-2xl leading-relaxed">{description}</p>
                )}
            </div>
            {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
    )
}
