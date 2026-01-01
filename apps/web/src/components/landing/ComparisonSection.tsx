export function ComparisonSection() {
    return (
        <section className="py-24 px-6 bg-background border-t border-dashed border-border/40">
            <div className="mx-auto max-w-6xl">
                <div className="flex flex-col items-center mb-16 text-center">
                    <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                        Philosophy
                    </span>
                    <h2 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-4 text-foreground">
                        Why we built this.
                    </h2>
                    <p className="text-muted-foreground text-sm font-medium max-w-xl">
                        The email industry has drifted towards marketing bloat and complexity. We are correcting course.
                    </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    <PhilosophyCard
                        tag="Economics"
                        problem="The 'Domain Tax' kills indie projects. Most providers charge per domain or force upgrades, meaning you have to shut down old ideas to start new ones."
                        solutionHeader="Unlimited Domains."
                        solutionText="Spin up a new project every weekend. We don't care how many domains you have. We only count the emails you actually send."
                    />

                    <PhilosophyCard
                        tag="Architecture"
                        problem="Inbound is usually an afterthought—paid add-ons, complex MX logic, and parsing raw MIME yourself."
                        solutionHeader="Native Threading."
                        solutionText="Send Sync to grab the `Message-ID` instantly for threading. Listen for replies via standard Webhooks or our type-safe `onReceive` callback."
                    />

                    <PhilosophyCard
                        tag="Billing"
                        problem="You pay for 'contacts', 'audiences', and marketing dashboards you never touch. You are subsidizing their marketing suite."
                        solutionHeader="Volume Only."
                        solutionText="Just the API. No bloat. $0.25/1k whether you send 10 or 10 million. You only pay for what flows through the pipe."
                    />

                    <PhilosophyCard
                        tag="Privacy"
                        problem="Your data is often stored, indexed, and potentially used to train their AI models."
                        solutionHeader="Scan & Drop."
                        solutionText="We act as a pipe, not a bucket. We scan your emails for safety and deliver them. We don't store your user content. Fully GDPR compliant."
                    />
                </div>
            </div>
        </section>
    )
}

function PhilosophyCard({
    tag,
    problem,
    solutionHeader,
    solutionText
}: {
    tag: string,
    problem: string,
    solutionHeader: string,
    solutionText: string
}) {
    return (
        <div className="group relative flex flex-col justify-between p-8 rounded-xl border border-dashed border-border/40 bg-background hover:bg-secondary/5 transition-colors overflow-hidden">
            <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:24px_24px] pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity" />

            <div className="flex flex-col gap-6 relative z-10">
                <span className="inline-block px-2 py-0.5 text-[9px] font-bold uppercase tracking-[0.2em] text-muted-foreground/50 border border-dashed border-border/40 w-fit rounded-sm">
                    {tag}
                </span>

                <div className="space-y-4">
                    <div className="pl-4 border-l-2 border-dashed border-border/40">
                        <p className="text-xs font-mono text-muted-foreground/60 leading-relaxed max-w-md">
                            <span className="uppercase tracking-wider text-[9px] font-bold text-muted-foreground/30 block mb-1">Industry Standard</span>
                            "{problem}"
                        </p>
                    </div>

                    <div className="space-y-2 mt-2">
                        <h3 className="text-2xl font-bold tracking-tight text-foreground">
                            {solutionHeader}
                        </h3>
                        <p className="text-sm font-medium text-muted-foreground leading-relaxed">
                            {solutionText}
                        </p>
                    </div>
                </div>
            </div>
        </div>
    )
}
