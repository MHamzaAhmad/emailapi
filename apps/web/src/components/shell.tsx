import { Link, useRouterState } from '@tanstack/react-router'
import { UserButton, useUser, SignedIn } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    ArrowRight01Icon,
    PackageIcon,
    Home01Icon,
    GlobeIcon,
    Key01Icon,
    WebhookIcon,
    Book02Icon,
    Settings01Icon
} from '@hugeicons/core-free-icons'
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbList,
    BreadcrumbPage,
    BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

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
    const { user } = useUser()
    const routerState = useRouterState()
    const currentPath = routerState.location.pathname

    const breadcrumbs = currentPath.split('/').filter(Boolean).map((segment, index, array) => {
        const href = `/${array.slice(0, index + 1).join('/')}`
        const isLast = index === array.length - 1
        const title = segment.charAt(0).toUpperCase() + segment.slice(1).replace(/-/g, ' ')

        return { title, href, isLast }
    })

    return (
        <div className="min-h-screen bg-background flex flex-col">
            {/* Top Navigation Bar */}
            <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur-md supports-[backdrop-filter]:bg-background/60">
                <div className="mx-auto flex h-14 max-w-7xl items-center px-4 md:px-6 gap-6">
                    <Link to="/" className="flex items-center gap-2.5 font-bold tracking-tight mr-4">
                        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
                            <HugeiconsIcon icon={PackageIcon} size={16} />
                        </div>
                        <span className="hidden md:inline-block">emailapi</span>
                    </Link>

                    <nav className="flex items-center gap-1">
                        {mainNav.map((item) => {
                            const isActive = currentPath === item.href || (item.href !== '/dashboard' && currentPath.startsWith(item.href))
                            return (
                                <Link
                                    key={item.href}
                                    to={item.href}
                                    className={`
                                        flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-all
                                        ${isActive
                                            ? 'bg-secondary text-foreground shadow-sm'
                                            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                                        }
                                    `}
                                >
                                    {/* <HugeiconsIcon icon={item.icon} size={14} className={isActive ? 'text-foreground' : ''} /> */}
                                    <span>{item.title}</span>
                                </Link>
                            )
                        })}
                    </nav>

                    <div className="flex-1" />

                    <div className="flex items-center gap-4">
                        <Button variant="ghost" size="sm" className="hidden md:flex gap-2 text-muted-foreground" asChild>
                            <Link to="/docs">
                                <HugeiconsIcon icon={Book02Icon} size={14} />
                                <span>Docs</span>
                            </Link>
                        </Button>
                        <Button variant="ghost" size="sm" className="hidden md:flex gap-2 text-muted-foreground w-8 px-0" asChild>
                            <Link to="/settings">
                                <HugeiconsIcon icon={Settings01Icon} size={16} />
                            </Link>
                        </Button>
                        <Separator orientation="vertical" className="h-4" />
                        <UserButton
                            appearance={{
                                elements: {
                                    avatarBox: "h-8 w-8 rounded-lg"
                                }
                            }}
                        />
                    </div>
                </div>
            </header>

            {/* Breadcrumbs & Content */}
            <main className="flex-1">
                <div className="mx-auto max-w-7xl px-4 md:px-6 py-6">
                    {breadcrumbs.length > 0 && (
                        <div className="mb-6">
                            <Breadcrumb>
                                <BreadcrumbList>
                                    <BreadcrumbItem>
                                        <BreadcrumbLink asChild>
                                            <Link to="/dashboard">Dashboard</Link>
                                        </BreadcrumbLink>
                                    </BreadcrumbItem>
                                    {breadcrumbs.length > 0 && breadcrumbs[0].href !== '/dashboard' && <BreadcrumbSeparator />}

                                    {breadcrumbs.filter(b => b.href !== '/dashboard').map((crumb) => (
                                        <BreadcrumbItem key={crumb.href}>
                                            {crumb.isLast ? (
                                                <BreadcrumbPage>{crumb.title}</BreadcrumbPage>
                                            ) : (
                                                <>
                                                    <BreadcrumbLink asChild>
                                                        <Link to={crumb.href}>{crumb.title}</Link>
                                                    </BreadcrumbLink>
                                                    <BreadcrumbSeparator />
                                                </>
                                            )}
                                        </BreadcrumbItem>
                                    ))}
                                </BreadcrumbList>
                            </Breadcrumb>
                        </div>
                    )}

                    <div className="animate-in fade-in slide-in-from-bottom-2 duration-500">
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
