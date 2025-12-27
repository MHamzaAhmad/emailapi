import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { cn } from "@/lib/utils";

type Mode = "async" | "sync";

export function PerformanceChart() {
    const [activeMode, setActiveMode] = useState<Mode>("async");

    return (
        <div className="w-full max-w-4xl mx-auto">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-12 md:gap-20 items-start">
                {/* Left: Interactive Comparison Table */}
                <div className="flex flex-col gap-8">
                    <div className="flex flex-col">
                        {/* Async Row */}
                        <button
                            onClick={() => setActiveMode("async")}
                            className={cn(
                                "group flex justify-between items-baseline border-b border-dashed border-border/40 pb-4 pt-2 transition-all duration-300 text-left",
                                activeMode === "async" ? "opacity-100 scale-[1.02] pl-2 border-primary/40 bg-secondary/10" : "opacity-60 hover:opacity-80"
                            )}
                        >
                            <div className="flex flex-col">
                                <span className={cn("text-sm font-medium transition-colors", activeMode === "async" ? "text-primary" : "text-foreground")}>
                                    Async API
                                </span>
                                {activeMode === "async" && (
                                    <motion.span
                                        layoutId="active-indicator"
                                        className="text-[10px] text-muted-foreground font-medium mt-1"
                                    >
                                        Fire & Forget
                                    </motion.span>
                                )}
                            </div>
                            <span className={cn("text-3xl font-bold tracking-tight tabular-nums transition-colors", activeMode === "async" ? "text-primary" : "text-foreground")}>
                                34ms
                            </span>
                        </button>

                        {/* Sync Row */}
                        <button
                            onClick={() => setActiveMode("sync")}
                            className={cn(
                                "group flex justify-between items-baseline border-b border-dashed border-border/40 pb-4 pt-4 transition-all duration-300 text-left",
                                activeMode === "sync" ? "opacity-100 scale-[1.02] pl-2 border-primary/40 bg-secondary/10" : "opacity-60 hover:opacity-80"
                            )}
                        >
                            <div className="flex flex-col">
                                <span className={cn("text-sm font-medium transition-colors", activeMode === "sync" ? "text-primary" : "text-foreground")}>
                                    Sync API
                                </span>
                                {activeMode === "sync" && (
                                    <motion.span
                                        layoutId="active-indicator"
                                        className="text-[10px] text-muted-foreground font-medium mt-1"
                                    >
                                        Wait for Upstream
                                    </motion.span>
                                )}
                            </div>
                            <span className={cn("text-xl font-semibold tracking-tight tabular-nums transition-colors", activeMode === "sync" ? "text-primary" : "text-muted-foreground")}>
                                122ms
                            </span>
                        </button>

                        {/* Legacy Row (Static) */}
                        <div className="flex justify-between items-baseline border-b border-dashed border-border/40 pb-4 pt-4 opacity-40 cursor-not-allowed">
                            <span className="text-sm font-medium text-muted-foreground">Legacy Providers</span>
                            <span className="text-xl font-semibold tracking-tight text-muted-foreground tabular-nums">~400ms</span>
                        </div>
                    </div>

                    {/* Dynamic Explanation Box */}
                    <div className="relative h-24">
                        <AnimatePresence mode="wait">
                            {activeMode === "async" ? (
                                <motion.div
                                    key="async"
                                    initial={{ opacity: 0, y: 10 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    exit={{ opacity: 0, y: -10 }}
                                    transition={{ duration: 0.2 }}
                                    className="absolute inset-0"
                                >
                                    <h4 className="text-xs font-bold uppercase tracking-wider text-foreground mb-2">Why use Async?</h4>
                                    <p className="text-sm text-muted-foreground leading-relaxed">
                                        Maximum throughput. We queue the email and handle retries reliably. Perfect for <span className="text-foreground font-medium">notifications, newsletters, and bulk sending</span>.
                                    </p>
                                </motion.div>
                            ) : (
                                <motion.div
                                    key="sync"
                                    initial={{ opacity: 0, y: 10 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    exit={{ opacity: 0, y: -10 }}
                                    transition={{ duration: 0.2 }}
                                    className="absolute inset-0"
                                >
                                    <h4 className="text-xs font-bold uppercase tracking-wider text-foreground mb-2">Why use Sync?</h4>
                                    <p className="text-sm text-muted-foreground leading-relaxed">
                                        Immediate feedback. We wait for the upstream provider to accept the message and return the <code className="bg-secondary px-1 py-0.5 rounded text-xs">Message-ID</code> instantly. Essential for <span className="text-foreground font-medium">threading and saving replies</span>.
                                    </p>
                                </motion.div>
                            )}
                        </AnimatePresence>
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
