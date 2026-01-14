import { HugeiconsIcon } from '@hugeicons/react'
import { Alert02Icon, Cancel01Icon } from '@hugeicons/core-free-icons'
import { useSuspensionStatus } from '@/hooks'
import { useState } from 'react'
import { InfoTooltip } from '@/components/ui/info-tooltip'

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

    // Simplified copy
    const title = isSuspended ? 'Account Suspended' : 'Usage Limit Warning'
    const description = isSuspended
        ? 'Your account has been suspended due to policy violations.'
        : 'Your account is under review due to unusual activity.'

    // Support link URL
    const supportEmail = "support@simpleemailapi.dev"
    const supportSubject = isSuspended ? "Account Suspension Appeal" : "Account Review Inquiry"

    return (
        <div className={`
            w-full px-4 py-3 border-b flex items-center justify-between gap-4
            ${isSuspended
                ? 'bg-destructive/10 border-destructive/20 text-destructive'
                : 'bg-amber-500/10 border-amber-500/20 text-amber-700 dark:text-amber-400'
            }
        `}>
            <div className="flex flex-col sm:flex-row items-center gap-2 sm:gap-4 mx-auto max-w-7xl w-full text-center sm:text-left">
                <div className="flex items-center gap-2 justify-center sm:justify-start">
                    <HugeiconsIcon
                        icon={Alert02Icon}
                        size={16}
                        strokeWidth={2.5}
                        className={isSuspended ? 'text-destructive' : 'text-amber-600 dark:text-amber-400'}
                    />
                    <span className="font-bold text-xs uppercase tracking-wider">{title}</span>
                </div>

                <div className="flex-1 flex flex-col sm:flex-row items-center gap-1 sm:gap-2 text-xs justify-center sm:justify-start">
                    <span className="opacity-90">{description}</span>
                    <div className="flex items-center gap-2">
                        <a
                            href={`mailto:${supportEmail}?subject=${encodeURIComponent(supportSubject)}`}
                            className="font-medium underline underline-offset-2 hover:opacity-80 transition-opacity whitespace-nowrap"
                        >
                            Contact support
                        </a>
                        {reason && (
                            <InfoTooltip content={reason}>
                                <HugeiconsIcon
                                    icon={Alert02Icon}
                                    size={12}
                                    className="opacity-60 hover:opacity-100 transition-opacity cursor-help"
                                />
                            </InfoTooltip>
                        )}
                    </div>
                </div>

                {!isSuspended && (
                    <button
                        onClick={() => setDismissed(true)}
                        className="absolute right-4 top-3 sm:static p-1 hover:bg-black/10 dark:hover:bg-white/10 rounded transition-colors"
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
