import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { SignedIn, SignedOut } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  ArrowRight01Icon,
  ZapIcon,
  CodeIcon,
  ShieldIcon,
} from '@hugeicons/core-free-icons'
import { Button } from '@/components/ui/button'
import { useEffect } from 'react'

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
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border">
        <div className="mx-auto max-w-5xl flex items-center justify-between px-6 h-12">
          <div className="flex items-center gap-1.5">
            <div className="h-5 w-5 rounded border border-foreground/30 flex items-center justify-center">
              <HugeiconsIcon icon={ArrowRight01Icon} size={12} strokeWidth={1.5} />
            </div>
            <span className="text-sm font-medium tracking-tight">emailapi</span>
          </div>
          <div className="flex items-center gap-3">
            <Link to="/docs">
              <Button variant="ghost" size="sm">Docs</Button>
            </Link>
            <Link to="/sign-in">
              <Button variant="ghost" size="sm">Sign in</Button>
            </Link>
            <Link to="/sign-up">
              <Button size="sm">Get Started</Button>
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto max-w-5xl px-6 py-24">
        <div className="max-w-2xl">
          <h1 className="text-3xl font-medium tracking-tight leading-tight">
            The simplest email API
          </h1>
          <p className="mt-3 text-base text-muted-foreground leading-relaxed">
            Send transactional emails with a single API call. No complexity, no bloat.
            Built for developers who want to ship.
          </p>
          <div className="mt-6 flex items-center gap-3">
            <Link to="/sign-up">
              <Button>
                Get your API key
                <HugeiconsIcon icon={ArrowRight01Icon} size={12} strokeWidth={1.5} />
              </Button>
            </Link>
            <Link to="/docs">
              <Button variant="outline">Read the docs</Button>
            </Link>
          </div>
        </div>

        {/* Code example */}
        <div className="mt-12 rounded-md border border-border bg-card p-4 font-mono text-xs">
          <div className="text-muted-foreground mb-2"># Send an email</div>
          <div>
            <span className="text-muted-foreground">curl</span>{' '}
            <span className="text-foreground">-X POST https://api.emailapi.dev/v1/send \</span>
          </div>
          <div className="pl-4">
            <span className="text-muted-foreground">-H</span>{' '}
            <span className="text-foreground">"Authorization: Bearer em_live_..." \</span>
          </div>
          <div className="pl-4">
            <span className="text-muted-foreground">-d</span>{' '}
            <span className="text-foreground">'{`{"from":"you@example.com","to":["user@example.com"],"subject":"Hello","body":"World"}`}'</span>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="border-t border-border">
        <div className="mx-auto max-w-5xl px-6 py-16">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <FeatureCard
              icon={ZapIcon}
              title="Fast"
              description="Sub-100ms API response times. Emails delivered in seconds."
            />
            <FeatureCard
              icon={CodeIcon}
              title="Simple"
              description="One endpoint to send emails. Native SDKs for TypeScript and Go."
            />
            <FeatureCard
              icon={ShieldIcon}
              title="Reliable"
              description="Automatic retries, bounce handling, and webhook delivery."
            />
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border">
        <div className="mx-auto max-w-5xl px-6 py-6">
          <p className="text-2xs text-muted-foreground">
            Email API
          </p>
        </div>
      </footer>
    </div>
  )
}

function FeatureCard({
  icon,
  title,
  description
}: {
  icon: typeof ZapIcon
  title: string
  description: string
}) {
  return (
    <div className="space-y-2">
      <div className="text-foreground">
        <HugeiconsIcon icon={icon} size={16} strokeWidth={1.5} />
      </div>
      <h3 className="text-sm font-medium">{title}</h3>
      <p className="text-xs text-muted-foreground leading-relaxed">{description}</p>
    </div>
  )
}
