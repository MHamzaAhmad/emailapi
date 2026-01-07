import { createFileRoute, Link, Outlet } from '@tanstack/react-router'
import { Logo } from '@/components/logo'
import { ModeToggle } from '@/components/mode-toggle'

export const Route = createFileRoute('/blog')({
    component: BlogLayout,
})

function BlogLayout() {
    return (
        <div className="min-h-screen bg-background text-foreground font-sans antialiased selection:bg-primary/20">
            {/* Background Grid */}
            <div className="fixed inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />

            {/* Navigation */}
            <nav className="border-b border-dashed border-border/40 bg-background/80 backdrop-blur-xl sticky top-0 z-50">
                <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-6">
                    <div className="flex items-center gap-6">
                        <Link to="/" className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:opacity-80 transition-opacity">
                            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                                <Logo className="h-3.5 w-3.5" />
                            </div>
                            <span className="text-sm">SimpleEmailAPI</span>
                        </Link>
                        <span className="text-border/40">/</span>
                        <Link to="/blog" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
                            Blog
                        </Link>
                    </div>
                    <div className="flex items-center gap-4">
                        <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">
                            Docs
                        </a>
                        <div className="h-4 w-[1px] bg-border/60" />
                        <ModeToggle />
                    </div>
                </div>
            </nav>

            {/* Content Outlet */}
            <Outlet />

            {/* Footer */}
            <footer className="border-t border-dashed border-border/40 bg-background py-14 px-6">
                <div className="mx-auto max-w-7xl flex flex-col md:flex-row justify-between items-center gap-8">
                    <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/80">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-secondary text-foreground shadow-sm ring-1 ring-black/5 dark:ring-white/10">
                            <Logo className="h-3.5 w-3.5" />
                        </div>
                        <span className="text-xs">SimpleEmailAPI</span>
                    </div>

                    <div className="flex items-center gap-6 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                        <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Documentation</a>
                        <span className="text-border/40">|</span>
                        <Link to="/blog" className="hover:text-foreground transition-colors">Blog</Link>
                        <span className="text-border/40">|</span>
                        <Link to="/terms" className="hover:text-foreground transition-colors">Terms</Link>
                        <span className="text-border/40">|</span>
                        <Link to="/privacy" className="hover:text-foreground transition-colors">Privacy</Link>
                        <span className="text-border/40">|</span>
                        <Link to="/about" className="hover:text-foreground transition-colors">About</Link>
                        <span className="text-border/40">|</span>
                        <p>&copy; {new Date().getFullYear()} SimpleEmailAPI</p>
                    </div>

                    <div className="flex gap-2 items-center px-3 py-1 rounded-md border border-dashed border-border/60 bg-secondary/30">
                        <div className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse ring-2 ring-emerald-500/20"></div>
                        <span className="text-[9px] font-bold uppercase tracking-[0.2em] text-muted-foreground">Systems Nominal</span>
                    </div>
                </div>
            </footer>
        </div>
    )
}
