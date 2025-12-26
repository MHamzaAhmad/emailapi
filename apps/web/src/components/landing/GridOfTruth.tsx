export function GridOfTruth() {
    return (
        <section className="py-24 px-6 md:px-8 bg-background border-t border-dashed border-border/40">
            <div className="mx-auto max-w-6xl">
                <div className="grid grid-cols-1 md:grid-cols-2">
                    {/* Performance */}
                    <div className="p-10 border-b border-dashed md:border-b-0 md:border-r border-border/40 flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Performance
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Sub-10ms <br /> Overhead.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                Built on HTTP/2 with a custom high-concurrency engine. We've optimized every byte of the delivery pipeline to ensure your emails reach their destination without delay.
                            </p>
                        </div>
                    </div>

                    {/* Affordability */}
                    <div className="p-10 flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Pricing
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                $0.25 <br /> per 1k emails.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                Transparent pricing. No monthly minimums. No hidden fees. Pay only for the infrastructure you consume, with the scale of a global enterprise.
                            </p>
                        </div>
                    </div>

                    {/* Privacy */}
                    <div className="p-10 border-t border-dashed border-border/40 md:border-r flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Integrity
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Private <br /> by Default.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                We do not store or read your email content. All data is processed in-memory and discarded upon delivery. GDPR and HIPAA compliant architecture.
                            </p>
                        </div>
                    </div>

                    {/* Developer Experience */}
                    <div className="p-10 border-t border-dashed border-border/40 flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Engineering
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Native <br /> SDKs.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                Ship faster with first-class support for TypeScript, Go, and Rust. Or use our simple REST API. Designed by engineers, for engineers.
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    )
}
