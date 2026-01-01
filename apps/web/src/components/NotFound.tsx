import { Link } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import { AlertCircleIcon, Home01Icon } from '@hugeicons/core-free-icons'

export function NotFound() {
    return (
        <div className="min-h-screen grid place-items-center bg-background px-6">
            {/* Background Grid */}
            <div className="fixed inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />

            <div className="w-full max-w-md p-8 rounded-xl border border-dashed border-red-500/20 bg-red-500/5 relative overflow-hidden text-center">
                {/* Decorative corner markers */}
                <div className="absolute top-0 left-0 w-2 h-2 border-t border-l border-red-500/40" />
                <div className="absolute top-0 right-0 w-2 h-2 border-t border-r border-red-500/40" />
                <div className="absolute bottom-0 left-0 w-2 h-2 border-b border-l border-red-500/40" />
                <div className="absolute bottom-0 right-0 w-2 h-2 border-b border-r border-red-500/40" />

                <div className="flex justify-center mb-6 text-red-500/80">
                    <HugeiconsIcon icon={AlertCircleIcon} size={48} strokeWidth={1.5} />
                </div>

                <h1 className="text-4xl font-mono font-bold tracking-tighter text-foreground mb-2">
                    404
                </h1>
                <h2 className="text-sm font-bold uppercase tracking-widest text-red-500/60 mb-6">
                    Signal Lost
                </h2>

                <p className="text-muted-foreground text-sm mb-8 leading-relaxed">
                    The requested coordinate does not exist in this sector.
                </p>

                <Link
                    to="/"
                    className="inline-flex items-center gap-2 h-9 px-4 rounded-md bg-background border border-dashed border-border hover:border-foreground/20 hover:bg-secondary/50 transition-all text-xs font-medium text-foreground group"
                >
                    <HugeiconsIcon icon={Home01Icon} size={14} className="text-muted-foreground group-hover:text-foreground transition-colors" />
                    Return to Console
                </Link>
            </div>
        </div>
    )
}
