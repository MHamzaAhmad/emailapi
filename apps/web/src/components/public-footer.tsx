import { Link } from '@tanstack/react-router'
import { Logo } from '@/components/logo'

export function PublicFooter() {
    return (
        <footer className="border-t border-dashed border-border/40 bg-background py-14 px-6 mb-20 md:mb-0">
            <div className="mx-auto max-w-7xl flex flex-col md:flex-row justify-between items-center gap-8">
                {/* Logo and Copyright */}
                <div className="flex flex-col md:flex-row items-center gap-4 md:gap-8">
                    <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/80">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-secondary text-foreground shadow-sm ring-1 ring-black/5 dark:ring-white/10">
                            <Logo className="h-3.5 w-3.5" />
                        </div>
                        <span className="text-xs">SimpleEmailAPI</span>
                    </div>
                    <p className="text-xs text-muted-foreground">&copy; {new Date().getFullYear()} SimpleEmailAPI</p>
                </div>

                {/* Navigation */}
                <nav className="flex flex-wrap items-center justify-center gap-6 text-sm font-medium text-muted-foreground">
                    <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">
                        Docs
                    </a>
                    <Link to="/blog" className="hover:text-foreground transition-colors">Blog</Link>
                    <Link to="/terms" className="hover:text-foreground transition-colors">Terms</Link>
                    <Link to="/privacy" className="hover:text-foreground transition-colors">Privacy</Link>
                    <Link to="/about" className="hover:text-foreground transition-colors">About</Link>
                </nav>
            </div>
        </footer>
    )
}
