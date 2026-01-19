import { motion } from "framer-motion";

export function PerformanceChart() {
    return (
        <div className="w-full max-w-4xl mx-auto">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-12 md:gap-20 items-start">
                {/* Left: Performance Comparison */}
                <div className="flex flex-col gap-8">
                    <div className="flex flex-col">
                        {/* SimpleEmailAPI Row - Highlighted */}
                        <div
                            className="group flex justify-between items-baseline border-b border-dashed border-primary/40 pb-4 pt-2 pl-2 bg-secondary/10 scale-[1.02]"
                        >
                            <div className="flex flex-col">
                                <span className="text-sm font-medium text-primary">
                                    SimpleEmailAPI
                                </span>
                                <motion.span
                                    initial={{ opacity: 0, y: 5 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    className="text-[10px] text-muted-foreground font-medium mt-1"
                                >
                                    Async-first • Fire & Forget
                                </motion.span>
                            </div>
                            <span className="text-3xl font-bold tracking-tight tabular-nums text-primary">
                                34ms
                            </span>
                        </div>

                        {/* Legacy Row (Static) */}
                        <div className="flex justify-between items-baseline border-b border-dashed border-border/40 pb-4 pt-4 opacity-40 cursor-not-allowed">
                            <span className="text-sm font-medium text-muted-foreground">Legacy Providers</span>
                            <span className="text-xl font-semibold tracking-tight text-muted-foreground tabular-nums">~400ms</span>
                        </div>
                    </div>

                    {/* Explanation Box */}
                    <div>
                        <h4 className="text-xs font-bold uppercase tracking-wider text-foreground mb-2">Async by Default</h4>
                        <p className="text-sm text-muted-foreground leading-relaxed">
                            Every email is processed asynchronously with automatic retries and delivery tracking. Queue and go — we handle the rest. Perfect for <span className="text-foreground font-medium">transactional emails, notifications, and bulk sending</span>.
                        </p>
                    </div>
                </div>

                {/* Right: Stats & High Level Tech */}
                <div className="flex flex-col justify-center gap-8">
                    <div className="grid grid-cols-2 gap-6">
                        <div className="flex flex-col gap-1 p-4 rounded-lg border border-dashed border-border/40 bg-background/50">
                            <span className="text-3xl sm:text-4xl font-bold tracking-tight text-foreground">
                                34<span className="text-base text-muted-foreground align-top ml-1">ms</span>
                            </span>
                            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                Global Latency
                            </span>
                        </div>
                        <div className="flex flex-col gap-1 p-4 rounded-lg border border-dashed border-border/40 bg-background/50">
                            <span className="text-3xl sm:text-4xl font-bold tracking-tight text-foreground">
                                100<span className="text-base text-muted-foreground align-top ml-1">+</span>
                            </span>
                            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                RPS / Conn
                            </span>
                        </div>
                        <div className="flex flex-col gap-1 p-4 rounded-lg border border-dashed border-border/40 bg-background/50">
                            <span className="text-3xl sm:text-4xl font-bold tracking-tight text-foreground">
                                99.9<span className="text-base text-muted-foreground align-top ml-1">%</span>
                            </span>
                            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                Uptime
                            </span>
                        </div>
                        <div className="flex flex-col gap-1 p-4 rounded-lg border border-dashed border-border/40 bg-background/50">
                            <span className="text-3xl sm:text-4xl font-bold tracking-tight text-foreground">
                                gRPC
                            </span>
                            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                Native Support
                            </span>
                        </div>
                    </div>

                    <div className="space-y-4">
                        <h3 className="text-lg font-semibold text-foreground">Engineered for high throughput.</h3>
                        <p className="text-sm text-muted-foreground leading-relaxed">
                            Whether you choose REST or type-safe gRPC, our edge-optimized network processes your emails in milliseconds, not seconds.
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
}
