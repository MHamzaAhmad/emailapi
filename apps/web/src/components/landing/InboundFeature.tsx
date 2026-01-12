import { CodeWindow } from '@/components/ui/code-window'
import { HugeiconsIcon } from '@hugeicons/react'
import { ArrowRight01Icon, SparklesIcon } from '@hugeicons/core-free-icons'

export function InboundFeature() {
    return (
        <section className="py-24 px-6 md:px-8 bg-background border-t border-dashed border-border/40">
            <div className="mx-auto max-w-4xl text-center">

                {/* Centered Header */}
                <div className="mb-12 flex flex-col items-center">
                    <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                        Native Inbound
                    </span>
                    <h2 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-6 text-foreground leading-[1.1]">
                        Receive as easily as you send.
                    </h2>
                    <p className="text-muted-foreground text-sm font-medium leading-relaxed max-w-lg mx-auto mb-8">
                        Most providers treat inbound as an afterthought. We don't.
                        <br />
                        Every email you send has a working <code className="text-xs bg-secondary px-1 py-0.5 rounded border border-border/40">Reply-To</code> that auto-routes to your webhook.
                    </p>

                    <a href="https://docs.simpleemailapi.dev/receiving/on-receive" className="inline-flex items-center gap-2 text-xs font-semibold text-primary hover:underline group">
                        <span>Read the Inbound Docs</span>
                        <HugeiconsIcon icon={ArrowRight01Icon} size={14} className="group-hover:translate-x-0.5 transition-transform" />
                    </a>
                </div>

                {/* Centered Code Window */}
                <div className="relative mx-auto max-w-2xl px-2 sm:px-0 text-left">
                    <div className="absolute -inset-10 bg-gradient-to-b from-primary/5 to-transparent blur-3xl -z-10 rounded-full opacity-50" />

                    <CodeWindow
                        className="w-full shadow-2xl border-border/60"
                        tabs={[{
                            label: "Webhook Payload",
                            value: "json",
                            language: "json",
                        }]}
                    />
                </div>

            </div>
        </section>
    )
}
