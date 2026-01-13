import { Link } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Menu02Icon,
    Book02Icon,
    Home01Icon,
    InformationCircleIcon,
    News01Icon,
    Legal01Icon,
    SecurityCheckIcon,
} from '@hugeicons/core-free-icons'
import { Button } from '@/components/ui/button'
import { ModeToggle } from '@/components/mode-toggle'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Logo } from '@/components/logo'
import {
    LazySignedIn,
    LazySignedOut,
    LazySignInButton,
    LazyUserButton,
} from '@/components/lazy-clerk'

export function PublicNavbar() {
    return (
        <nav className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
            <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 md:px-6">
                <div className="flex items-center gap-4">
                    {/* Mobile Menu - Left Aligned */}
                    <div className="md:hidden flex items-center gap-4">
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground">
                                    <HugeiconsIcon icon={Menu02Icon} size={20} />
                                    <span className="sr-only">Toggle menu</span>
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="start" className="w-[200px] ml-2">
                                <DropdownMenuItem asChild>
                                    <Link to="/" className="w-full cursor-pointer flex items-center gap-2">
                                        <HugeiconsIcon icon={Home01Icon} size={16} />
                                        <span>Home</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                    <Link to="/about" className="w-full cursor-pointer flex items-center gap-2">
                                        <HugeiconsIcon icon={InformationCircleIcon} size={16} />
                                        <span>About</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                    <Link to="/blog" className="w-full cursor-pointer flex items-center gap-2">
                                        <HugeiconsIcon icon={News01Icon} size={16} />
                                        <span>Blog</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                    <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="w-full cursor-pointer flex items-center gap-2">
                                        <HugeiconsIcon icon={Book02Icon} size={16} />
                                        <span>Documentation</span>
                                    </a>
                                </DropdownMenuItem>
                                <div className="h-[1px] bg-border/40 my-1" />
                                <DropdownMenuItem asChild>
                                    <Link to="/terms" className="w-full cursor-pointer flex items-center gap-2">
                                        <HugeiconsIcon icon={Legal01Icon} size={16} />
                                        <span>Terms</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                    <Link to="/privacy" className="w-full cursor-pointer flex items-center gap-2">
                                        <HugeiconsIcon icon={SecurityCheckIcon} size={16} />
                                        <span>Privacy</span>
                                    </Link>
                                </DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                        <div className="h-4 w-[1px] bg-border" />
                    </div>

                    <Link to="/" className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:text-foreground transition-colors">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                            <Logo className="h-3.5 w-3.5" />
                        </div>
                        <span className="text-sm hidden md:inline-block">SimpleEmailAPI</span>
                        <span className="hidden md:inline-block px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-primary bg-primary/10 rounded border border-primary/20">Beta</span>
                    </Link>
                </div>

                {/* Desktop Nav */}
                <div className="hidden md:flex items-center gap-4">
                    <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">Documentation</a>
                    <LazySignedOut>
                        <LazySignInButton mode="modal">
                            <Button size="sm" variant="outline" className="h-8 px-4 text-xs font-medium border-dashed border-border hover:bg-muted/50 rounded-md">
                                Login
                            </Button>
                        </LazySignInButton>
                    </LazySignedOut>
                    <LazySignedIn>
                        <LazyUserButton
                            appearance={{
                                elements: {
                                    avatarBox: "h-7 w-7 rounded-full ring-2 ring-background hover:ring-muted transition-all"
                                }
                            }}
                        />
                    </LazySignedIn>
                    <div className="h-4 w-[1px] bg-border/60 mx-1" />
                    <ModeToggle />
                </div>

                {/* Mobile Quick Actions (Right) */}
                <div className="md:hidden flex items-center gap-3">
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground" asChild>
                        <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer">
                            <HugeiconsIcon icon={Book02Icon} size={18} />
                        </a>
                    </Button>

                    <ModeToggle />

                    <div className="h-4 w-[1px] bg-border/60 mx-1" />

                    <LazySignedOut>
                        <LazySignInButton mode="modal">
                            <Button size="sm" variant="default" className="h-8 px-3 text-xs font-medium">
                                Login
                            </Button>
                        </LazySignInButton>
                    </LazySignedOut>
                    <LazySignedIn>
                        <LazyUserButton
                            appearance={{
                                elements: {
                                    avatarBox: "h-7 w-7 rounded-full ring-2 ring-background hover:ring-muted transition-all"
                                }
                            }}
                        />
                    </LazySignedIn>
                </div>
            </div>
        </nav>
    )
}
