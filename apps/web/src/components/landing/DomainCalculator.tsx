import { useState } from 'react'

export function DomainCalculator() {
    const [domains, setDomains] = useState(3)

    // Pricing Logic (Competitors)
    const getCompetitorPrice = (n: number) => {
        if (n <= 1) return 0
        if (n <= 10) return 20
        return 90
    }

    const competitorPrice = getCompetitorPrice(domains)
    const yearlySavings = competitorPrice * 12

    return (
        <section className="py-24 px-6 md:px-8 bg-background border-t border-dashed border-border/40">
            <div className="mx-auto max-w-2xl"> {/* Compact width */}

                {/* Minimal Header */}
                <div className="text-center mb-10">
                    <span className="inline-block px-2 py-0.5 mb-4 text-[9px] font-bold uppercase tracking-[0.2em] text-muted-foreground/50 border border-dashed border-border/60 rounded-sm">
                        2026 Pricing Reality
                    </span>
                    <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-4">
                        Stop paying per domain.
                    </h2>
                    <p className="text-muted-foreground text-sm font-medium leading-relaxed max-w-xl mx-auto">
                        Other providers penalize you for having multiple projects. We charge you for volume, not for creativity.
                    </p>
                </div>

                {/* Compact Calculator Widget */}
                <div className="rounded-xl border border-dashed border-border/60 bg-muted/5 relative overflow-hidden">

                    {/* Interactive Area */}
                    <div className="p-6 sm:p-8">
                        {/* Slider Label */}
                        <div className="flex justify-between items-end mb-6">
                            <label className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                                How many domains do you have?
                            </label>
                            <span className="text-2xl font-mono font-bold text-foreground">
                                {domains}
                            </span>
                        </div>

                        {/* Slider */}
                        <input
                            type="range"
                            min="1"
                            max="20"
                            step="1"
                            value={domains}
                            onChange={(e) => setDomains(Number(e.target.value))}
                            style={{ accentColor: "hsl(var(--primary))" }}
                            className="w-full h-1.5 bg-secondary border border-border/50 rounded-lg appearance-none cursor-pointer mb-2"
                        />
                        <div className="flex justify-between text-[9px] font-mono text-muted-foreground/40">
                            <span>1</span>
                            <span>20+</span>
                        </div>
                    </div>

                    {/* Results Split - High Contrast */}
                    <div className="px-6 py-6 border-t border-dashed border-border/60 bg-background/50 backdrop-blur-sm grid grid-cols-2 gap-8 items-center">

                        {/* Competitors (Pain) */}
                        <div className="text-center opacity-70 grayscale transition-all duration-500" style={{ opacity: domains > 1 ? 1 : 0.5, filter: domains > 1 ? 'none' : 'grayscale(100%)' }}>
                            <div className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1">
                                Competitors
                            </div>
                            <div className="flex items-center justify-center gap-0.5 text-red-600">
                                <span className="text-lg font-bold">$</span>
                                <span className="text-3xl font-bold tracking-tighter">{competitorPrice}</span>
                                <span className="text-xs font-medium text-muted-foreground/60">/mo</span>
                            </div>
                            {domains > 1 && (
                                <span className="inline-block mt-2 text-[9px] font-medium text-red-600/80 bg-red-600/10 px-1.5 py-0.5 rounded">
                                    {domains <= 10 ? 'Includes 1 domain limit' : 'Includes 5-10 domain limit'}
                                </span>
                            )}
                        </div>

                        {/* Us (Gain) */}
                        <div className="text-center relative">
                            <div className="absolute -inset-4 bg-primary/5 blur-xl rounded-full -z-10" />
                            <div className="text-[10px] font-bold uppercase tracking-wider text-primary mb-1">
                                SimpleEmailAPI
                            </div>
                            <div className="flex items-center justify-center gap-0.5 text-foreground">
                                <span className="text-lg font-bold">$</span>
                                <span className="text-3xl font-bold tracking-tighter">0</span>
                                <span className="text-xs font-medium text-muted-foreground/60">/mo</span>
                            </div>
                            <span className="inline-block mt-2 text-[9px] font-medium text-primary/80 px-1.5 py-0.5 rounded border border-dashed border-primary/20">
                                Unlimited Domains
                            </span>
                        </div>
                    </div>

                    {/* Footer - The Conclusion */}
                    <div className={`
                        py-3 px-6 text-center border-t border-dashed border-border/60 transition-colors duration-300
                        ${yearlySavings > 0 ? 'bg-primary/5' : 'bg-muted/10'}
                    `}>
                        <p className="text-xs font-medium text-foreground">
                            {yearlySavings > 0 ? (
                                <>
                                    You save <span className="font-bold text-primary">${yearlySavings}/year</span> switching to us.
                                </>
                            ) : (
                                <span className="text-muted-foreground">Even with 1 domain, you get more features for free.</span>
                            )}
                        </p>
                    </div>

                </div>
            </div>
        </section>
    )
}
