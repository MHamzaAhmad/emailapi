import { createFileRoute } from '@tanstack/react-router'
import { PackageIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

export const Route = createFileRoute('/terms')({
    component: TermsPage,
})

function TermsPage() {
    return (
        <div className="min-h-screen bg-background text-foreground font-sans antialiased selection:bg-primary/20">
            {/* Background Grid */}
            <div className="fixed inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />

            {/* Navigation Stub */}
            <nav className="border-b border-dashed border-border/40 bg-background/80 backdrop-blur-xl sticky top-0 z-50">
                <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-6">
                    <a href="/" className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:opacity-80 transition-opacity">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                            <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
                        </div>
                        <span className="text-sm">SimpleEmailAPI</span>
                    </a>
                </div>
            </nav>

            <main className="mx-auto max-w-3xl px-6 py-24">
                <div className="mb-12">
                    <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                        Legal
                    </span>
                    <h1 className="text-4xl sm:text-5xl font-bold tracking-tighter mb-4 text-foreground">
                        Terms of Service
                    </h1>
                    <p className="text-muted-foreground font-mono text-sm">
                        Last Updated: January 1, 2026
                    </p>
                </div>

                <div className="prose prose-sm prose-invert max-w-none text-muted-foreground">
                    <p className="text-base leading-relaxed mb-8">
                        Welcome to simpleemailapi.dev. By using our API and services, you agree to comply with and be bound by the following terms.
                    </p>

                    <Section number="01" title="Acceptance of Terms">
                        <p>
                            By accessing or using simpleemailapi.dev, you agree to be bound by these Terms of Service and the AWS Acceptable Use Policy (AUP). If you are using the service on behalf of an organization, you agree to these terms for that organization.
                        </p>
                    </Section>

                    <Section number="02" title="Account Security and API Keys">
                        <p>
                            You are responsible for maintaining the confidentiality of your API keys.
                        </p>
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40">
                            <li>Any activity that occurs under your API key is your sole responsibility.</li>
                            <li>You must notify us immediately of any unauthorized use of your account.</li>
                        </ul>
                    </Section>

                    <Section number="03" title="Acceptable Use & Prohibited Content">
                        <p>You agree NOT to use simpleemailapi.dev to:</p>
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40">
                            <li>Send "Cold Emails," unsolicited bulk email (SPAM), or marketing to purchased/scraped lists.</li>
                            <li>Distribute malware, viruses, or any malicious code.</li>
                            <li>Engage in phishing, identity theft, or fraudulent activity.</li>
                            <li>Send content that is illegal, defamatory, or violates intellectual property rights.</li>
                        </ul>
                    </Section>

                    <Section number="04" title="Mandatory Recipient Consent">
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40">
                            <li>You may only send emails to recipients who have explicitly opted-in to receive communication from you.</li>
                            <li>You must include a functional Unsubscribe link in all marketing/newsletter communications.</li>
                            <li>Our system automatically manages a Suppression List. If a recipient unsubscribes or files a complaint, you are prohibited from attempting to contact them again via our API.</li>
                        </ul>
                    </Section>

                    <Section number="05" title="Reputation Monitoring and Suspension">
                        <p>
                            To protect the integrity of our platform, we monitor sending reputation in real-time. We reserve the right to throttle or suspend your account without notice if:
                        </p>
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40">
                            <li>Your Bounce Rate exceeds 5%.</li>
                            <li>Your Complaint Rate exceeds 0.1%.</li>
                            <li>You are found to be in violation of the AWS Acceptable Use Policy.</li>
                        </ul>
                    </Section>

                    <Section number="06" title="Disclaimer of Warranties">
                        <p>
                            simpleemailapi.dev is provided "as is." While we strive for 100% uptime and high deliverability, we do not guarantee that our service will be uninterrupted or error-free. We are not responsible for any messages blocked or filtered by receiving mail servers.
                        </p>
                    </Section>

                    <Section number="07" title="Limitation of Liability">
                        <p>
                            In no event shall simpleemailapi.dev or its owner be liable for any indirect, incidental, or consequential damages resulting from the use or inability to use the service.
                        </p>
                    </Section>

                    <Section number="08" title="Termination">
                        <p>
                            We reserve the right to terminate your access to the API at any time, for any reason, particularly for violations of our anti-spam policies.
                        </p>
                    </Section>

                </div>
            </main>

            {/* Footer */}
            <footer className="border-t border-dashed border-border/40 bg-background py-14 px-6 text-center">
                <p className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                    &copy; {new Date().getFullYear()} SimpleEmailAPI
                </p>
            </footer>
        </div>
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
            <div className="text-sm text-muted-foreground leading-7">
                {children}
            </div>
        </section>
    )
}
