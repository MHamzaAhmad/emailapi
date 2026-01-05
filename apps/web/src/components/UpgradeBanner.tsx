import { Link } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'

interface UpgradeBannerProps {
    plan: string
    emailsSent?: number
    limit?: number
}

export function UpgradeBanner({ plan, emailsSent = 0, limit = 3000 }: UpgradeBannerProps) {
    if (plan !== 'free') {
        return null
    }

    const percentage = Math.min((emailsSent / limit) * 100, 100)
    const isNearLimit = percentage >= 80

    return (
        <div className="mb-6 rounded-lg border border-dashed border-border/60 bg-secondary/20 px-4 py-3">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                <div className="flex-1">
                    <p className="text-xs font-medium text-foreground mb-1">
                        Free Plan
                    </p>
                    <div className="flex items-center gap-3">
                        <div className="flex-1 max-w-[200px] h-1.5 rounded-full bg-border/40 overflow-hidden">
                            <div
                                className={`h-full rounded-full transition-all ${isNearLimit ? 'bg-amber-500' : 'bg-primary'
                                    }`}
                                style={{ width: `${percentage}%` }}
                            />
                        </div>
                        <span className="text-[10px] font-medium text-muted-foreground">
                            {emailsSent.toLocaleString()} / {limit.toLocaleString()} emails
                        </span>
                    </div>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    className="h-7 px-3 text-xs font-medium border-dashed"
                    asChild
                >
                    <Link to="/settings/billing">
                        Upgrade
                    </Link>
                </Button>
            </div>
        </div>
    )
}
