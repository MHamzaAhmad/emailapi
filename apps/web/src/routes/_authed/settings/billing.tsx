import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { queryKeys } from '@/lib/queryClient'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { HugeiconsIcon } from '@hugeicons/react'
import { HelpCircleIcon, Tick02Icon, Settings02Icon, CreditCardIcon } from '@hugeicons/core-free-icons'
import { usePlans, useCurrentSubscription, useCreateCheckoutSession, useGetCustomerPortalUrl } from '@/hooks'

export const Route = createFileRoute('/_authed/settings/billing')({
    validateSearch: (search: Record<string, unknown>): { success?: boolean } => ({
        success: search.success === 'true',
    }),
    component: BillingSettingsPage,
})

function BillingSettingsPage() {
    const { data: plansData, isLoading: plansLoading } = usePlans()
    const { data: subscription } = useCurrentSubscription()
    const createCheckout = useCreateCheckoutSession()
    const getPortal = useGetCustomerPortalUrl()

    const queryClient = useQueryClient()
    const { success } = Route.useSearch()

    const plans = plansData?.plans || []
    const currentPlan = subscription?.planId || 'free'
    const isPaid = currentPlan !== 'free'
    const hasPolarCustomer = Boolean(subscription?.polarCustomerId)

    const handleUpgrade = async (planId: string) => {
        if (isPaid) {
            await handleOpenPortal()
            return
        }

        try {
            const result = await createCheckout.mutateAsync({
                planId,
                successUrl: `${window.location.origin}/settings/billing?success=true`,
            })
            window.location.href = result.checkoutUrl
        } catch (error) {
            console.error('Failed to create checkout session:', error)
        }
    }

    const handleOpenPortal = async () => {
        try {
            const result = await getPortal.mutateAsync()
            window.location.href = result.portalUrl
        } catch (error) {
            console.error('Failed to get portal URL:', error)
        }
    }

    // Invalidate cache if we just returned from a successful checkout
    useEffect(() => {
        if (success) {
            void queryClient.invalidateQueries({ queryKey: ['billing', 'subscription'] })
            void queryClient.invalidateQueries({ queryKey: queryKeys.users.me() })

            // Clean up the URL without reloading
            const newUrl = window.location.pathname
            window.history.replaceState({}, '', newUrl)
        }
    }, [success, queryClient])

    const formatPrice = (priceCents: bigint | number): string => {
        const dollars = Number(priceCents) / 100
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD',
            minimumFractionDigits: 0,
            maximumFractionDigits: 2,
        }).format(dollars)
    }

    if (plansLoading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-sm text-muted-foreground">Loading plans...</div>
            </div>
        )
    }

    const currentPlanInfo = plans.find(p => p.id === currentPlan)

    return (
        <div className="flex flex-col gap-6 mb-8 mt-2">
            {/* Header / Actions Section */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                {/* Left Side - Title/Context */}
                <div className="flex items-center gap-3">
                    {/* Placeholder for future filters or just empty to push actions right, 
                        or we can keep the title if the user prefers, but dashboard often keeps it clean. 
                        Let's add a subtle label like the dashboard's filter row if needed, 
                        but effectively we just want the actions on the right mostly. */}
                </div>

                {/* Right Side - Actions */}
                <div className="flex items-center gap-3 w-full sm:w-auto justify-end">
                    <div className="flex items-center gap-2 h-8 px-3 rounded-md border border-dashed border-border/60 bg-muted/20 text-xs text-muted-foreground font-medium select-none">
                        <span>Current Plan:</span>
                        <span className="text-foreground flex items-center gap-1.5">
                            {currentPlanInfo?.name || 'Free'}
                            {currentPlan !== 'free' && (
                                <span className="flex h-1.5 w-1.5 rounded-full bg-primary" />
                            )}
                        </span>
                    </div>

                    {hasPolarCustomer && (
                        <Button
                            variant="outline"
                            size="sm"
                            className="h-8 gap-2 text-xs font-medium shadow-sm bg-background hover:bg-muted/50 border-dashed"
                            onClick={handleOpenPortal}
                            disabled={getPortal.isPending}
                        >
                            <HugeiconsIcon icon={Settings02Icon} size={14} />
                            {getPortal.isPending ? 'Loading...' : 'Manage Subscription'}
                        </Button>
                    )}
                </div>
            </div>

            {/* Plans Grid Container - Matching Dashboard Card Style */}
            <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-0 rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden divide-y md:divide-y-0 md:divide-x divide-border/40">
                    {plans.map((plan) => (
                        <div
                            key={plan.id}
                            className={`
                                p-6 flex flex-col relative group
                                ${plan.id === currentPlan ? 'bg-secondary/30' : 'hover:bg-muted/30 transition-colors'}
                            `}
                        >
                            {plan.id === currentPlan && (
                                <div className="absolute top-4 right-4 text-primary opacity-50">
                                    <HugeiconsIcon icon={Tick02Icon} size={16} strokeWidth={2.5} />
                                </div>
                            )}

                            <div className="mb-6 space-y-1">
                                <h4 className="font-bold text-sm tracking-tight">{plan.name}</h4>
                                <div className="flex items-baseline gap-1">
                                    <span className="text-2xl font-bold tracking-tight">{formatPrice(plan.priceCents)}</span>
                                    <span className="text-xs text-muted-foreground font-medium">/mo</span>
                                </div>
                                <p className="text-[11px] text-muted-foreground leading-relaxed pt-1 min-h-[2.5em]">
                                    {plan.description}
                                </p>
                            </div>

                            <div className="flex-1 mb-6">
                                <ul className="space-y-3">
                                    {plan.features.map((feature, idx) => (
                                        <li key={idx} className="flex items-start gap-2.5 text-xs text-muted-foreground/80">
                                            <div className="mt-1 h-3 w-3 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                                                <div className="h-1 w-1 rounded-full bg-primary/60" />
                                            </div>
                                            <div className="flex items-center gap-1.5 flex-1 leading-snug">
                                                <span>{feature.name}</span>
                                                {feature.tooltip && (
                                                    <Tooltip>
                                                        <TooltipTrigger className="cursor-help flex items-center opacity-50 hover:opacity-100 transition-opacity">
                                                            <HugeiconsIcon icon={HelpCircleIcon} size={12} />
                                                        </TooltipTrigger>
                                                        <TooltipContent className="max-w-[220px] text-[10px] leading-tight p-3">
                                                            {feature.tooltip.split('\n\n').map((part, i) => (
                                                                <div key={i} className={i > 0 ? "mt-2" : ""}>
                                                                    {part.split(/(\*\*.*?\*\*)/).map((segment, j) => {
                                                                        if (segment.startsWith('**') && segment.endsWith('**')) {
                                                                            return <span key={j} className="font-semibold text-foreground">{segment.slice(2, -2)}</span>
                                                                        }
                                                                        return segment
                                                                    })}
                                                                </div>
                                                            ))}
                                                        </TooltipContent>
                                                    </Tooltip>
                                                )}
                                            </div>
                                        </li>
                                    ))}
                                </ul>
                            </div>

                            <div className="mt-auto pt-2">
                                {plan.id !== currentPlan && plan.id !== 'free' ? (
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        className="w-full h-8 text-xs font-medium border-dashed hover:border-solid hover:bg-primary hover:text-primary-foreground hover:border-primary transition-all shadow-sm"
                                        onClick={() => handleUpgrade(plan.id)}
                                        disabled={createCheckout.isPending || getPortal.isPending}
                                    >
                                        <HugeiconsIcon icon={isPaid ? Settings02Icon : CreditCardIcon} size={14} className="mr-2" />
                                        {createCheckout.isPending || getPortal.isPending ? 'Processing...' : (isPaid ? 'Switch Plan' : 'Upgrade Plan')}
                                    </Button>
                                ) : plan.id === currentPlan ? (
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="w-full h-8 text-xs font-medium bg-muted/50 cursor-default text-muted-foreground hover:bg-muted/50 border border-transparent"
                                        disabled
                                    >
                                        Current Plan
                                    </Button>
                                ) : (
                                    <div className="h-8" /> /* Spacer for free plan if not current */
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            <div className="text-center text-[10px] text-muted-foreground/40 pt-6 uppercase tracking-widest font-medium">
                Payments secured by Polar.sh
            </div>
        </div>
    )
}
