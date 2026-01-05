import { createFileRoute } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Shell, PageHeader } from '@/components/shell'
import { usePlans, useCurrentSubscription, useCreateCheckoutSession, useGetCustomerPortalUrl } from '@/hooks'

export const Route = createFileRoute('/_authed/settings/billing')({
    component: BillingSettingsPage,
})

function BillingSettingsPage() {
    const { data: plansData, isLoading: plansLoading } = usePlans()
    const { data: subscription } = useCurrentSubscription()
    const createCheckout = useCreateCheckoutSession()
    const getPortal = useGetCustomerPortalUrl()

    const plans = plansData?.plans || []
    const currentPlan = subscription?.plan || 'free'
    const hasPolarCustomer = Boolean(subscription?.polarCustomerId)

    const handleUpgrade = async (planId: string) => {
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

    const formatPrice = (priceCents: bigint): string => {
        const dollars = Number(priceCents) / 100
        return dollars === 0 ? '$0' : `$${dollars}`
    }

    const formatLimit = (limit: bigint): string => {
        if (limit === BigInt(-1)) return 'Unlimited'
        return Number(limit).toLocaleString()
    }

    if (plansLoading) {
        return (
            <Shell>
                <PageHeader title="Billing" description="Manage your subscription and billing settings." />
                <div className="flex items-center justify-center h-64">
                    <div className="text-sm text-muted-foreground">Loading plans...</div>
                </div>
            </Shell>
        )
    }

    const currentPlanInfo = plans.find(p => p.id === currentPlan)

    return (
        <Shell>
            <PageHeader
                title="Billing"
                description="Manage your subscription and billing settings."
            />

            {/* Current Plan */}
            <div className="mb-8">
                <h3 className="text-sm font-medium mb-4">Current Plan</h3>
                <div className="rounded-lg border border-dashed border-border/60 bg-secondary/20 p-6">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-lg font-bold">
                                {currentPlanInfo?.name || 'Free'}
                            </p>
                            <p className="text-sm text-muted-foreground">
                                {currentPlanInfo?.description || 'Perfect for getting started'}
                            </p>
                        </div>
                        <div className="text-right">
                            <p className="text-2xl font-bold">
                                {currentPlanInfo ? formatPrice(currentPlanInfo.priceCents) : '$0'}
                                <span className="text-sm text-muted-foreground ml-1">/mo</span>
                            </p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Plan Comparison */}
            <div className="mb-8">
                <h3 className="text-sm font-medium mb-4">Available Plans</h3>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-0 border border-dashed border-border/60 rounded-lg overflow-hidden">
                    {plans.map((plan, i) => (
                        <div
                            key={plan.id}
                            className={`
                                p-6 flex flex-col
                                ${i < plans.length - 1 ? 'border-b md:border-b-0 md:border-r border-dashed border-border/40' : ''}
                                ${plan.id === currentPlan ? 'bg-secondary/30' : ''}
                            `}
                        >
                            <div className="mb-4">
                                <div className="flex items-center gap-2 mb-1">
                                    <h4 className="text-sm font-bold">{plan.name}</h4>
                                    {plan.id === currentPlan && (
                                        <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground bg-secondary px-1.5 py-0.5 rounded">
                                            Current
                                        </span>
                                    )}
                                </div>
                                <p className="text-xs text-muted-foreground">{plan.description}</p>
                            </div>

                            <div className="mb-4">
                                <span className="text-2xl font-bold">{formatPrice(plan.priceCents)}</span>
                                <span className="text-xs text-muted-foreground ml-1">
                                    {plan.id === 'payg' ? '/1K emails' : '/mo'}
                                </span>
                            </div>

                            <ul className="space-y-2 mb-6 flex-1">
                                <li className="flex items-start gap-2 text-xs text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>{formatLimit(plan.monthlyLimit)} emails / month</span>
                                </li>
                                <li className="flex items-start gap-2 text-xs text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>
                                        {plan.dailyLimit === BigInt(-1)
                                            ? 'No daily limit'
                                            : `${formatLimit(plan.dailyLimit)} emails / day`}
                                    </span>
                                </li>
                                <li className="flex items-start gap-2 text-xs text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>Unlimited domains</span>
                                </li>
                            </ul>

                            {plan.id !== currentPlan && plan.id !== 'free' && (
                                <Button
                                    variant="outline"
                                    size="sm"
                                    className="w-full h-8 text-xs font-medium border-dashed"
                                    onClick={() => handleUpgrade(plan.id)}
                                    disabled={createCheckout.isPending}
                                >
                                    {createCheckout.isPending ? 'Loading...' : 'Upgrade'}
                                </Button>
                            )}
                        </div>
                    ))}
                </div>
            </div>

            {/* Manage Subscription */}
            {hasPolarCustomer && (
                <div className="mb-8">
                    <h3 className="text-sm font-medium mb-4">Manage Subscription</h3>
                    <div className="rounded-lg border border-dashed border-border/60 p-6">
                        <p className="text-sm text-muted-foreground mb-4">
                            Manage your subscription, update payment methods, and view invoices in the customer portal.
                        </p>
                        <Button
                            variant="outline"
                            size="sm"
                            className="h-8 text-xs font-medium border-dashed"
                            onClick={handleOpenPortal}
                            disabled={getPortal.isPending}
                        >
                            {getPortal.isPending ? 'Loading...' : 'Open Customer Portal'}
                        </Button>
                    </div>
                </div>
            )}
        </Shell>
    )
}
