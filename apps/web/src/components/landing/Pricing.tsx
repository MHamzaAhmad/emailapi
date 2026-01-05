import { SignInButton, SignedIn, SignedOut } from '@clerk/clerk-react'
import { Link } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { usePlans } from '@/hooks'

export function Pricing() {
    const { data: plansData, isLoading } = usePlans()
    const plans = plansData?.plans || []

    const formatPrice = (priceCents: bigint): string => {
        const dollars = Number(priceCents) / 100
        return dollars === 0 ? '$0' : `$${dollars}`
    }

    const formatLimit = (limit: bigint): string => {
        if (limit === BigInt(-1)) return 'Unlimited'
        return Number(limit).toLocaleString()
    }

    if (isLoading || plans.length === 0) {
        // Fallback for SSR / loading state
        return null
    }

    return (
        <section className="py-24 px-6 bg-background border-t border-dashed border-border/40">
            <div className="mx-auto max-w-6xl">
                <div className="flex flex-col items-center mb-16 text-center">
                    <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                        Pricing
                    </span>
                    <h2 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-4 text-foreground">
                        Simple, Transparent Pricing.
                    </h2>
                    <p className="text-muted-foreground text-sm font-medium max-w-xl">
                        Start free, scale when you're ready. No hidden fees, no surprises.
                    </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-0 border border-dashed border-border/60 rounded-lg overflow-hidden">
                    {plans.map((plan, i) => (
                        <div
                            key={plan.id}
                            className={`
                                p-8 flex flex-col
                                ${i < plans.length - 1 ? 'border-b md:border-b-0 md:border-r border-dashed border-border/40' : ''}
                                ${plan.id === 'scale' ? 'bg-secondary/20' : ''}
                            `}
                        >
                            <div className="mb-6">
                                <div className="flex items-center gap-2 mb-1">
                                    <h3 className="text-lg font-bold">{plan.name}</h3>
                                    {plan.id === 'scale' && (
                                        <span className="text-[9px] font-bold uppercase tracking-wider text-primary bg-primary/10 px-1.5 py-0.5 rounded">
                                            Popular
                                        </span>
                                    )}
                                </div>
                                <p className="text-sm text-muted-foreground">{plan.description}</p>
                            </div>

                            <div className="mb-6">
                                <span className="text-4xl font-bold">{formatPrice(plan.priceCents)}</span>
                                <span className="text-sm text-muted-foreground ml-1">
                                    {plan.id === 'payg' ? '/1K emails' : '/mo'}
                                </span>
                            </div>

                            <ul className="space-y-3 mb-8 flex-1">
                                <li className="flex items-start gap-2 text-sm text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>{formatLimit(plan.monthlyLimit)} emails / month</span>
                                </li>
                                <li className="flex items-start gap-2 text-sm text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>
                                        {plan.dailyLimit === BigInt(-1)
                                            ? 'No daily limit'
                                            : `${formatLimit(plan.dailyLimit)} emails / day`}
                                    </span>
                                </li>
                                <li className="flex items-start gap-2 text-sm text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>Unlimited domains</span>
                                </li>
                                <li className="flex items-start gap-2 text-sm text-muted-foreground">
                                    <span className="text-foreground/60 mt-0.5">—</span>
                                    <span>Webhook events</span>
                                </li>
                                {plan.id !== 'free' && (
                                    <li className="flex items-start gap-2 text-sm text-muted-foreground">
                                        <span className="text-foreground/60 mt-0.5">—</span>
                                        <span>{plan.id === 'payg' ? 'Enterprise support' : 'Priority support'}</span>
                                    </li>
                                )}
                            </ul>

                            <SignedOut>
                                <SignInButton mode="modal">
                                    <Button
                                        variant={plan.id === 'scale' ? 'default' : 'outline'}
                                        size="sm"
                                        className={`w-full h-9 text-xs font-medium ${plan.id !== 'scale' ? 'border-dashed' : ''}`}
                                    >
                                        Get Started
                                    </Button>
                                </SignInButton>
                            </SignedOut>
                            <SignedIn>
                                <Button
                                    variant={plan.id === 'scale' ? 'default' : 'outline'}
                                    size="sm"
                                    className={`w-full h-9 text-xs font-medium ${plan.id !== 'scale' ? 'border-dashed' : ''}`}
                                    asChild
                                >
                                    <Link to="/settings/billing">
                                        {plan.id === 'free' ? 'Current Plan' : 'Upgrade'}
                                    </Link>
                                </Button>
                            </SignedIn>
                        </div>
                    ))}
                </div>

                <p className="text-center text-xs text-muted-foreground mt-8">
                    All plans include unlimited domains and webhook integrations.
                    <br />
                    Need more? <a href="mailto:support@simpleemailapi.dev" className="underline hover:text-foreground transition-colors">Contact us</a> for enterprise pricing.
                </p>
            </div>
        </section>
    )
}
