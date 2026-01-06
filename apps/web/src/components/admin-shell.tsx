import { Link, useRouterState } from '@tanstack/react-router'
import { UserButton } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Setting07Icon,
    UserMultiple02Icon,
    Alert02Icon,
    ArrowLeft01Icon,
} from '@hugeicons/core-free-icons'
import { ModeToggle } from '@/components/mode-toggle'

interface AdminShellProps {
    children: React.ReactNode
}

const adminNav = [
    { title: 'Users', href: '/admin/users', icon: UserMultiple02Icon },
    { title: 'Flagged', href: '/admin/flagged', icon: Alert02Icon },
]

export function AdminShell({ children }: AdminShellProps) {
    const routerState = useRouterState()
    const currentPath = routerState.location.pathname

    return (
        <div className="min-h-screen bg-background flex flex-col font-sans">
            {/* Admin Navigation Bar */}
            <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
                <div className="mx-auto flex h-14 max-w-7xl items-center px-4 md:px-6 gap-4">
                    {/* Back to Dashboard */}
                    <Link
                        to="/dashboard"
                        className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors mr-2"
                    >
                        <HugeiconsIcon icon={ArrowLeft01Icon} size={16} />
                        <span className="text-xs hidden sm:inline">Dashboard</span>
                    </Link>

                    <div className="h-4 w-[1px] bg-border/60" />

                    {/* Admin Badge */}
                    <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 mr-6">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-destructive text-destructive-foreground shadow-sm">
                            <HugeiconsIcon icon={Setting07Icon} size={14} strokeWidth={2.5} />
                        </div>
                        <span className="hidden md:inline-block text-sm">Admin Console</span>
                    </div>

                    {/* Admin Nav */}
                    <nav className="flex items-center gap-1">
                        {adminNav.map((item) => {
                            const isActive = currentPath === item.href || currentPath.startsWith(item.href + '/')
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
                                    <HugeiconsIcon icon={item.icon} size={14} />
                                    <span>{item.title}</span>
                                </Link>
                            )
                        })}
                    </nav>

                    <div className="flex-1" />

                    <div className="flex items-center gap-3">
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

            {/* Content */}
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

export default AdminShell
