import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { SignedIn, SignedOut } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  ArrowRight01Icon,
  ZapIcon,
  CodeIcon,
  ShieldIcon,
  CheckmarkCircle01Icon,
} from '@hugeicons/core-free-icons'
import { useEffect } from 'react'
import { Button } from '@/components/ui/button'

export const Route = createFileRoute('/')(
  {
    component: LandingPage,
  }
)

function LandingPage() {
  return (
    <>
      <SignedIn>
        <RedirectToDashboard />
      </SignedIn>
      <SignedOut>
        <LandingContent />
      </SignedOut>
    </>
  )
}

function RedirectToDashboard() {
  const navigate = useNavigate()

  useEffect(() => {
    navigate({ to: '/dashboard' })
  }, [navigate])

  return null
}

function LandingContent() {
  return (
    <div className="min-h-screen bg-background text-foreground font-sans selection:bg-foreground selection:text-background flex flex-col">
      {/* Navigation */}
      <nav className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-6">
          <div className="flex items-center gap-2">
            <div className="flex h-6 w-6 items-center justify-center rounded-sm bg-primary text-primary-foreground">
              <HugeiconsIcon icon={ArrowRight01Icon} size={14} strokeWidth={2.5} />
            </div>
            <span className="text-sm font-bold tracking-tight">emailapi.dev</span>
          </div>
          <div className="hidden md:flex items-center gap-6">
            <Link to="/docs" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">Documentation</Link>
            <Link to="/sign-in" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">Login</Link>
            <Button asChild size="sm" className="h-8 font-semibold">
              <Link to="/sign-up">
                Get Started
              </Link>
            </Button>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <main className="flex-1">
        <section className="relative px-6 pt-32 pb-24 lg:pt-48 lg:pb-32 overflow-hidden">
          <div className="mx-auto max-w-3xl text-center">
            <div className="mb-8 inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium text-muted-foreground bg-muted/50 backdrop-blur-sm">
              <HugeiconsIcon icon={CheckmarkCircle01Icon} size={12} className="mr-2" />
              v1.0.0 Stable Release
            </div>
            <h1 className="mb-8 text-5xl font-bold tracking-tight sm:text-7xl">
              Zero-Bloat <br className="hidden sm:block" /> Email Delivery
            </h1>
            <p className="mx-auto mb-10 max-w-xl text-lg text-muted-foreground leading-relaxed">
              A minimal gRPC & REST API for developers. No dashboard clutter.
              No marketing bloat. Just direct-to-inbox speed.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <Button asChild size="lg" className="h-12 px-8 text-base">
                <Link to="/sign-up">Start Building Free</Link>
              </Button>
              <Button asChild variant="outline" size="lg" className="h-12 px-8 text-base bg-background/50 hover:bg-muted/50">
                <Link to="/docs">Read the Docs</Link>
              </Button>
            </div>
          </div>

          {/* Subtle Grid Background */}
          <div className="absolute inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />
        </section>

        {/* Code Example Section */}
        <section className="px-6 py-20 border-y bg-muted/10">
          <div className="mx-auto max-w-6xl grid grid-cols-1 lg:grid-cols-2 gap-16 items-center">
            <div>
              <h2 className="text-2xl font-bold tracking-tight mb-4">API-First Design</h2>
              <p className="text-muted-foreground leading-relaxed mb-6">
                Connect your application in seconds using our simplified SDKs or direct HTTP endpoints.
                We handle the delivery pipeline, you handle the product.
              </p>
              <ul className="space-y-4 text-sm font-medium text-muted-foreground">
                <li className="flex items-center gap-3">
                  <HugeiconsIcon icon={CheckmarkCircle01Icon} size={16} className="text-primary" />
                  <span>Type-safe SDKs for TS, Go, Python</span>
                </li>
                <li className="flex items-center gap-3">
                  <HugeiconsIcon icon={CheckmarkCircle01Icon} size={16} className="text-primary" />
                  <span>Idempotency keys built-in</span>
                </li>
                <li className="flex items-center gap-3">
                  <HugeiconsIcon icon={CheckmarkCircle01Icon} size={16} className="text-primary" />
                  <span>Webhooks for real-time events</span>
                </li>
              </ul>
            </div>
            <div className="relative overflow-hidden rounded-xl border bg-card/80 shadow-sm backdrop-blur">
              <div className="flex items-center gap-1.5 border-b p-4 bg-muted/40">
                <div className="h-2.5 w-2.5 rounded-full bg-red-500/20 border border-red-500/30" />
                <div className="h-2.5 w-2.5 rounded-full bg-yellow-500/20 border border-yellow-500/30" />
                <div className="h-2.5 w-2.5 rounded-full bg-green-500/20 border border-green-500/30" />
              </div>
              <div className="p-6 overflow-x-auto">
                <pre className="text-sm font-mono leading-relaxed">
                  <span className="text-purple-500">curl</span> -X POST https://api.emailapi.dev/v1/send \<br />
                  {"  "}-H <span className="text-green-500">"Authorization: Bearer em_live_..."</span> \<br />
                  {"  "}-d <span className="text-blue-500">'{`{`}'</span><br />
                  {"    "}<span className="text-foreground">"from"</span>: <span className="text-green-500">"ops@system.io"</span>,<br />
                  {"    "}<span className="text-foreground">"to"</span>: [<span className="text-green-500">"user@dest.com"</span>],<br />
                  {"    "}<span className="text-foreground">"subject"</span>: <span className="text-green-500">"System Alert"</span>,<br />
                  {"    "}<span className="text-foreground">"body"</span>: <span className="text-green-500">"Your build is complete."</span><br />
                  {"  "}<span className="text-blue-500">'{`}`}'</span>
                </pre>
              </div>
            </div>
          </div>
        </section>

        {/* Features Section */}
        <section className="py-24 px-6">
          <div className="mx-auto max-w-6xl">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              <FeatureItem
                icon={ZapIcon}
                title="Nano-Latency"
                description="Engineered for speed with a Go-based core and global edge delivery nodes."
              />
              <FeatureItem
                icon={CodeIcon}
                title="Developer Native"
                description="REST, gRPC, and GraphQL support. Type-safe SDKs for all major languages."
              />
              <FeatureItem
                icon={ShieldIcon}
                title="Secure by Default"
                description="Strict DKIM, SPF, and DMARC enforcement. Isolated infrastructure for every key."
              />
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t py-12 px-6 bg-muted/20">
        <div className="mx-auto max-w-6xl flex flex-col md:flex-row justify-between items-center gap-6">
          <div className="flex items-center gap-2">
            <div className="h-4 w-4 bg-foreground/20 rounded-sm" />
            <span className="text-xs font-bold tracking-widest text-muted-foreground uppercase">emailapi</span>
          </div>
          <p className="text-xs text-muted-foreground font-mono">© 2025 EMAILAPI ENGINE INC.</p>
        </div>
      </footer>
    </div>
  )
}

function FeatureItem({ icon, title, description }: { icon: any, title: string, description: string }) {
  return (
    <div className="group rounded-xl border bg-card p-8 transition-shadow hover:shadow-md">
      <div className="mb-4 inline-flex items-center justify-center rounded-lg border bg-background p-3 shadow-sm">
        <HugeiconsIcon icon={icon} size={20} strokeWidth={1.5} />
      </div>
      <h3 className="mb-2 text-sm font-bold tracking-tight">{title}</h3>
      <p className="text-sm text-muted-foreground leading-relaxed">{description}</p>
    </div>
  )
}
