import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_public/privacy')({
    component: PrivacyPage,
})

function PrivacyPage() {
    return (
        <>
            {/* Background Grid */}
            <div className="fixed inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />

            <main className="mx-auto max-w-3xl px-6 py-24">
                <div className="mb-12">
                    <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                        Legal
                    </span>
                    <h1 className="text-4xl sm:text-5xl font-bold tracking-tighter mb-4 text-foreground">
                        Privacy Policy
                    </h1>
                    <p className="text-muted-foreground font-mono text-sm">
                        Last Updated: January 1, 2026
                    </p>
                </div>

                <div className="prose prose-sm prose-invert max-w-none text-muted-foreground">
                    <p className="text-base leading-relaxed mb-8">
                        At simpleemailapi.dev, we prioritize the privacy and security of your data. This policy explains how we handle the information processed through our API.
                    </p>

                    <Section number="01" title="Our &quot;Privacy by Design&quot; Philosophy">
                        <p>
                            Unlike traditional email service providers, simpleemailapi.dev is designed to be a pass-through security layer. We do not store the content of your emails or your recipient databases. Our infrastructure is built to process, scan, and deliver—not to archive.
                        </p>
                    </Section>

                    <Section number="02" title="Information We Process">
                        <p>When you use our API to send an email, we process the following:</p>
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40 mt-2">
                            <li><strong className="text-foreground">Email Metadata:</strong> We temporarily process sender and recipient addresses to facilitate delivery and handle bounces/complaints via AWS SES.</li>
                            <li><strong className="text-foreground">Content Scanning:</strong> For the safety of the platform and the recipient, we scan email bodies and attachments.</li>
                            <li><strong className="text-foreground">Links:</strong> We check URLs against real-time blacklists to prevent phishing.</li>
                            <li><strong className="text-foreground">Attachments:</strong> We scan files for malicious signatures and prohibited file types.</li>
                            <li><strong className="text-foreground">Ephemeral Processing:</strong> Once the security scan is complete and the email is handed off to the delivery network, the email content is purged from our active memory.</li>
                        </ul>
                    </Section>

                    <Section number="03" title="Data We Store">
                        <p>We only store the minimum data necessary to operate the service:</p>
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40 mt-2">
                            <li><strong className="text-foreground">Account Data:</strong> Your email address, API keys, and usage metadata (counters for billing and rate limiting).</li>
                            <li><strong className="text-foreground">Suppression Data:</strong> We maintain a list of hashed email addresses that have unsubscribed or complained. This is required to ensure we do not contact them again in violation of anti-spam laws.</li>
                            <li><strong className="text-foreground">Security Logs:</strong> We log metadata about blocked attempts (e.g., if an email was rejected because it contained a virus) to help improve our security filters.</li>
                        </ul>
                    </Section>

                    <Section number="04" title="Data Sharing and Third Parties">
                        <p>We do not sell, rent, or trade your data. To deliver your emails, we share the necessary data with:</p>
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40 mt-2">
                            <li><strong className="text-foreground">Amazon Web Services (AWS):</strong> Our infrastructure provider and delivery engine.</li>
                            <li><strong className="text-foreground">Security Databases:</strong> We may check URLs/Hashes against third-party security databases to identify known threats.</li>
                        </ul>
                    </Section>

                    <Section number="05" title="Data Retention">
                        <ul className="list-disc pl-4 space-y-1 marker:text-muted-foreground/40">
                            <li><strong className="text-foreground">Email Content:</strong> Deleted immediately after processing and delivery.</li>
                            <li><strong className="text-foreground">Analytics:</strong> We store aggregate data (e.g., "Total emails sent in January") for billing purposes.</li>
                            <li><strong className="text-foreground">Suppression Lists:</strong> Retained indefinitely to honor unsubscribe requests.</li>
                        </ul>
                    </Section>

                    <Section number="06" title="Your Rights">
                        <p>
                            As a developer, you have full control over your account. You may delete your API key or account at any time, which will remove all associated account metadata from our systems.
                        </p>
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
            <div className="text-sm text-muted-foreground leading-7">
                {children}
            </div>
        </section>
    )
}
