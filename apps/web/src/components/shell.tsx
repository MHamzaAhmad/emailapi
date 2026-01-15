import { Link, useRouterState } from '@tanstack/react-router'
import { Logo } from '@/components/logo'
import { UserButton } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import { useCurrentSubscription } from '@/hooks'
import {
    Home01Icon,
    GlobeIcon,
    Key01Icon,
    WebhookIcon,
    Book02Icon,
    Settings02Icon,
    Menu02Icon,
} from '@hugeicons/core-free-icons'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
    const { data: subscription } = useCurrentSubscription()
    const hasActivePlan = subscription?.planId && subscription.planId !== 'free'

    return (
        <div className="min-h-screen bg-background flex flex-col font-sans">
            {/* Top Navigation Bar - Ultra Minimal */}
            <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
                <div className="mx-auto flex h-14 max-w-7xl items-center px-4 md:px-6 gap-4">
                    {/* Mobile Navigation Group */}
                    <div className="flex items-center gap-4 md:hidden mr-2">
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="h-8 w-8 -ml-2 text-muted-foreground hover:text-foreground">
                                    <HugeiconsIcon icon={Menu02Icon} size={20} />
                                    <span className="sr-only">Toggle menu</span>
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="start" className="w-[200px] ml-2">
                                {mainNav.map((item) => (
                                    <DropdownMenuItem key={item.href} asChild>
                                        <Link
                                            to={item.href}
                                            className="w-full cursor-pointer flex items-center gap-2"
                                        >
                                            <HugeiconsIcon icon={item.icon} size={16} />
                                            <span>{item.title}</span>
                                        </Link>
                                    </DropdownMenuItem>
                                ))}
                            </DropdownMenuContent>
                        </DropdownMenu>
                        <div className="h-4 w-[1px] bg-border" />
                    </div>

                    <Link to="/" className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:text-foreground transition-colors mr-6">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                            <Logo className="h-3.5 w-3.5" />
                        </div>
                        <span className="hidden md:inline-block text-sm">SimpleEmailAPI</span>
                    </Link>

                    <nav className="hidden md:flex items-center gap-1">
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
                            <Link to="https://docs.simpleemailapi.dev" target="_blank">
                                <HugeiconsIcon icon={Book02Icon} size={16} />
                            </Link>
                        </Button>
                        {hasActivePlan ? (
                            <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground" asChild>
                                <Link to="/settings/billing">
                                    <HugeiconsIcon icon={Settings02Icon} size={16} />
                                </Link>
                            </Button>
                        ) : (
                            <Button variant="default" size="sm" className="h-8 text-xs font-medium" asChild>
                                <Link to="/settings/billing">
                                    Upgrade
                                </Link>
                            </Button>
                        )}
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
