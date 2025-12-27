export function GridOfTruth() {
    return (
        <section className="py-24 px-6 md:px-8 bg-background border-t border-dashed border-border/40">
            <div className="mx-auto max-w-6xl">
                <div className="grid grid-cols-1 md:grid-cols-2">
                    {/* Inbound / Reply-Ready */}
                    <div className="p-10 border-b border-dashed md:border-b-0 md:border-r border-border/40 flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Inbound
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Every email <br /> is reply-ready.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                Why overengineer inbound? With us, every email you send has a working Reply-To. You just add standard MX records and listen for the webhook. No complex parsing logic. It just works.
                            </p>
                        </div>
                    </div>

                    {/* Simplicity / Anti-Bloat */}
                    <div className="p-10 flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Philosophy
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Just the API. <br /> No marketing fluff.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                We are not a marketing platform. We don't have audience segmentation or drag-and-drop builders. We just do one thing perfectly: deliver your transactional emails instantly.
                            </p>
                        </div>
                    </div>

                    {/* Pricing */}
                    <div className="p-10 border-t border-dashed border-border/40 md:border-r flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Value
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Pay for usage. <br /> Not for domains.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                Why pay $80/mo solely to add a second domain? We charge for volume: <span className="text-foreground">$0.25/1k emails</span>. Unlimited domains, unlimited seats, zero bloat.
                            </p>
                        </div>
                    </div>

                    {/* Performance / Latency */}
                    <div className="p-10 border-t border-dashed border-border/40 flex flex-col justify-between min-h-[300px]">
                        <div>
                            <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                                Control
                            </span>
                            <h3 className="text-3xl font-bold tracking-tighter mb-4 text-foreground leading-[1.1]">
                                Throughput <br /> or Threading.
                            </h3>
                            <p className="text-muted-foreground text-sm leading-relaxed max-w-sm font-medium">
                                Two modes, zero compromise. Use Async (34ms) for fire-and-forget blasts. Use Sync when you need the Message-ID instantly to thread replies. You choose per request.
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    )
}
