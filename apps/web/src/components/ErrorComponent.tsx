import { useRouter } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import { Alert02Icon, RefreshIcon } from '@hugeicons/core-free-icons'

export function ErrorComponent({ error }: { error: Error }) {
    const router = useRouter()

    return (
        <div className="min-h-screen grid place-items-center bg-background px-6">
            {/* Background Grid */}
            <div className="fixed inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />

            <div className="w-full max-w-lg p-8 rounded-xl border border-dashed border-red-500/20 bg-red-500/5 relative overflow-hidden text-center">
                <div className="flex justify-center mb-6 text-red-500/80">
                    <HugeiconsIcon icon={Alert02Icon} size={48} strokeWidth={1.5} />
                </div>

                <h1 className="text-xl font-bold tracking-tight text-foreground mb-2">
                    System Malfunction
                </h1>
                <h2 className="text-[10px] font-bold uppercase tracking-widest text-red-500/60 mb-6">
                    Critical Error
                </h2>

                <div className="bg-black/20 rounded-md p-4 mb-8 text-left border border-white/5 overflow-auto max-h-48">
                    <code className="text-xs font-mono text-red-300 break-all">
                        {error.message || 'Unknown system error occurred.'}
                    </code>
                </div>

                <button
                    onClick={() => {
                        router.invalidate()
                        window.location.reload()
                    }}
                    className="inline-flex items-center gap-2 h-9 px-6 rounded-md bg-foreground text-background hover:bg-foreground/90 transition-all text-xs font-bold"
                >
                    <HugeiconsIcon icon={RefreshIcon} size={14} />
                    Retry Connection
                </button>
            </div>
        </div>
    )
}
