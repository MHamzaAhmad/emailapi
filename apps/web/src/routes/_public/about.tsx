import { createFileRoute } from '@tanstack/react-router'
import { TwitterIcon, Linkedin02Icon, Mail01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

export const Route = createFileRoute('/_public/about')({
    component: AboutPage,
})

function AboutPage() {
    return (
        <>
            {/* Background Grid */}
            <div className="fixed inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />

            <main className="mx-auto max-w-3xl px-6 py-24">
                <div className="mb-12">
                    <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                        Story
                    </span>
                    <h1 className="text-4xl sm:text-5xl font-bold tracking-tighter mb-4 text-foreground">
                        Built by an Indie Hacker.
                    </h1>
                    <p className="text-muted-foreground font-mono text-sm">
                        For the love of simplicity.
                    </p>
                </div>

                <div className="prose prose-sm prose-invert max-w-none text-muted-foreground">
                    <div className="text-base leading-relaxed mb-12">
                        <p>
                            Hi, I'm <strong className="text-foreground">Hamza</strong>.
                        </p>
                        <p className="mt-4">
                            This platform started as an experiment. I found myself asking: <em>Why does every other email API provider seem so complex?</em> Why is everything over-engineered? Why is support for inbound email often an afterthought, or missing entirely?
                        </p>
                        <p className="mt-4">
                            These questions kept piling up.
                        </p>
                        <ul className="list-disc pl-4 space-y-2 marker:text-muted-foreground/40 mt-4">
                            <li>Why do they store my emails?</li>
                            <li>Why do I have to pay per domain?</li>
                            <li>Why isn't there a perfect, simple API for just <strong>sending and receiving</strong> emails?</li>
                        </ul>
                    </div>

                    <Section number="01" title="The Mission">
                        <p>
                            I started building SimpleEmailAPI with one goal in mind: <strong>to build the simplest possible API for email.</strong>
                        </p>
                        <p className="mt-4">
                            I wanted something super performant that handles delivery, unsubscribes, and the heavy lifting, so you can just send emails. No marketing bloat. No "audiences," "contacts," "data segments," or other complications you didn't ask for. Just a simple interface to send email. Simply.
                        </p>
                    </Section>

                    <Section number="02" title="For Indie Hackers">
                        <p>
                            As an indie hacker, I know the struggle of reaching Product-Market Fit (PMF). We experiment with multiple products, launch fast, and validate ideas. But with other platforms providing only one domain on a free account, we're stuck.
                        </p>
                        <p className="mt-4">
                            To launch a new project, we often have to kill the emails for a previous one. That means no more signups and finding hacky workarounds.
                        </p>
                        <p className="mt-4">
                            <strong>Why not just keep trying?</strong>
                        </p>
                        <p className="mt-4">
                            With SimpleEmailAPI, you can keep adding domains. We only charge for <strong>email volume</strong>. Whether you have one project or ten, you pay for what you use, not for the number of domains or marketing features you don't need.
                        </p>
                    </Section>

                    <Section number="03" title="Connect">
                        <p>
                            This platform is built by an indie hacker, for indie hackers. I'm always open to feedback, questions, or just a chat.
                        </p>
                        <div className="flex flex-col sm:flex-row gap-4 mt-6">
                            <SocialLink href="https://twitter.com/hamzadotsh" icon={TwitterIcon} label="@hamzadotsh" />
                            <SocialLink href="https://www.linkedin.com/in/mhamza88/" icon={Linkedin02Icon} label="M. Hamza Ahmad" />
                            <SocialLink href="mailto:hamza@simpleemailapi.dev" icon={Mail01Icon} label="hamza@simpleemailapi.dev" />
                        </div>
                    </Section>

                </div>
            </main>
        </>
    )
}

function Section({ number, title, children }: { number: string, title: string, children: React.ReactNode }) {
    return (
        <section className="mb-12 relative pl-8 border-l border-dashed border-border/40 group">
            <div className="absolute -left-[3px] top-0 h-1.5 w-1.5 rounded-full bg-border/60 group-hover:bg-primary transition-colors" />
            <h2 className="text-xl font-bold tracking-tight text-foreground mb-4 flex items-baseline gap-3">
                <span className="text-xs font-mono text-muted-foreground font-medium opacity-50">{number}</span>
                {title}
            </h2>
            <div className="text-base text-muted-foreground leading-relaxed">
                {children}
            </div>
        </section>
    )
}

function SocialLink({ href, icon, label }: { href: string, icon: any, label: string }) {
    return (
        <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 px-4 py-3 rounded-md border border-dashed border-border/60 bg-secondary/10 hover:bg-secondary/30 transition-colors group"
        >
            <HugeiconsIcon icon={icon} size={18} className="text-muted-foreground group-hover:text-foreground transition-colors" />
            <span className="text-sm font-medium text-foreground">{label}</span>
        </a>
    )
}
