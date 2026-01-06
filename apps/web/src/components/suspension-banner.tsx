import { HugeiconsIcon } from '@hugeicons/react'
import { Alert02Icon, Cancel01Icon } from '@hugeicons/core-free-icons'
import { useSuspensionStatus } from '@/hooks'
import { formatDate } from '@/lib/utils'
import { useState } from 'react'

/**
 * Suspension Banner - displays when user is suspended or flagged
 * Shows at the top of the authenticated layout
 */
export function SuspensionBanner() {
    const { data: status, isLoading } = useSuspensionStatus()
    const [dismissed, setDismissed] = useState(false)

    // Don't show while loading or if dismissed
    if (isLoading || dismissed) return null

    // Don't show if not suspended/flagged
    if (!status?.isSuspended && !status?.isFlagged) return null

    const isSuspended = status.isSuspended
    const reason = isSuspended ? status.suspensionReason : status.flaggedReason
    const suspendedAt = status.suspendedAt
        ? formatDate(new Date(Number(status.suspendedAt.seconds) * 1000))
        : null

    return (
        <div className={`
            w-full px-4 py-3 border-b flex items-center justify-between gap-4
            ${isSuspended
                ? 'bg-destructive/10 border-destructive/20 text-destructive'
                : 'bg-amber-500/10 border-amber-500/20 text-amber-700 dark:text-amber-400'
            }
        `}>
            <div className="flex items-center gap-3 mx-auto max-w-7xl flex-1">
                <HugeiconsIcon
                    icon={Alert02Icon}
                    size={18}
                    className={isSuspended ? 'text-destructive' : 'text-amber-600 dark:text-amber-400'}
                />
                <div className="flex-1">
                    <span className="font-semibold text-sm">
                        {isSuspended ? 'Account Suspended' : 'Account Under Review'}
                    </span>
                    {reason && (
                        <span className="text-sm ml-2 opacity-80">
                            — {reason}
                        </span>
                    )}
                    {isSuspended && suspendedAt && (
                        <span className="text-xs ml-2 opacity-60">
                            (since {suspendedAt})
                        </span>
                    )}
                </div>
                {!isSuspended && (
                    <button
                        onClick={() => setDismissed(true)}
                        className="p-1 hover:bg-black/10 dark:hover:bg-white/10 rounded transition-colors"
                        aria-label="Dismiss"
                    >
                        <HugeiconsIcon icon={Cancel01Icon} size={14} />
                    </button>
                )}
            </div>
        </div>
    )
}

export default SuspensionBanner
