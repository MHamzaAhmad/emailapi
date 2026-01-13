import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { lazy, Suspense, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import {
  LazySignedIn,
  LazySignedOut,
  LazySignInButton,
} from '@/components/lazy-clerk'

// CodeWindow is in hero (LCP element) - import directly for faster paint
import { CodeWindow } from '@/components/ui/code-window'

// Lazy-load below-fold sections
const DomainCalculator = lazy(() => import('@/components/landing/DomainCalculator').then(m => ({ default: m.DomainCalculator })))
const InboundFeature = lazy(() => import('@/components/landing/InboundFeature').then(m => ({ default: m.InboundFeature })))
const Pricing = lazy(() => import('@/components/landing/Pricing').then(m => ({ default: m.Pricing })))
const PerformanceChart = lazy(() => import('@/components/landing/PerformanceChart').then(m => ({ default: m.PerformanceChart })))

export const Route = createFileRoute('/_public/')(
  {
    component: LandingPage,
    // Cache at CDN for 1 hour, stale-while-revalidate for 1 day
    headers: () => ({
      'Cache-Control': 'public, max-age=3600, stale-while-revalidate=86400',
    }),
  }
)

function LandingPage() {
  return (
    <>
      <LazySignedIn>
        <RedirectToDashboard />
      </LazySignedIn>
      <LazySignedOut>
        <LandingContent />
      </LazySignedOut>
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
    <>
      {/* Hero Section */}
      <main className="flex-1 w-full overflow-hidden">
        <section className="relative px-4 sm:px-6 pt-32 sm:pt-24 pb-16 sm:pb-20 overflow-hidden border-b border-dashed border-border/40">
          <div className="mx-auto max-w-4xl text-center">
            <h1 className="mb-6 text-4xl sm:text-5xl md:text-7xl font-bold tracking-tighter leading-[1.1] text-foreground">
              The Email API for <br /> <span className="text-muted-foreground/40 font-medium">Indie Hackers.</span>
            </h1>
            <p className="mx-auto mb-10 text-sm sm:text-base text-muted-foreground leading-relaxed max-w-xl font-medium px-4">
              <span className="text-foreground font-bold">Unlimited domains.</span> Pay only for what you send.
              <br />
              Just <span className="text-foreground">$0.25 per 1,000 emails</span>. No monthly fees for side projects.
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-3 mb-16 sm:mb-24 px-4 w-full">
              <LazySignInButton mode="modal">
                <Button className="h-9 px-6 text-xs font-semibold shadow-sm rounded-md w-full sm:w-auto">
                  Start Building for Free
                </Button>
              </LazySignInButton>
              <Button asChild variant="outline" className="h-9 px-6 text-xs font-semibold bg-background hover:bg-muted/50 rounded-md border-dashed border-border w-full sm:w-auto">
                <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer">
                  Read the Docs
                </a>
              </Button>
            </div>

            {/* Centered Code Window */}
            <div className="relative mx-auto max-w-3xl px-2 sm:px-0">
              <div className="absolute -inset-10 rounded-[3rem] bg-primary/5 blur-3xl -z-10" />
              <CodeWindow
                className="w-full border-border/60 rounded-xl shadow-2xl"
                tabs={[
                  {
                    label: "TypeScript",
                    value: "ts",
                    language: "typescript",
                  },
                  {
                    label: "cURL",
                    value: "curl",
                    language: "bash",
                  },
                  {
                    label: "Go",
                    value: "go",
                    language: "go",
                  }
                ]}
              />
            </div>
          </div>

          {/* Background Grid - Subtle Dashed */}
          <div className="absolute inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />
        </section>

        {/* The Hook: Domain Tax Comparison */}
        <Suspense fallback={null}>
          <DomainCalculator />
        </Suspense>

        {/* The Enabler: Inbound Feature */}
        <Suspense fallback={null}>
          <InboundFeature />
        </Suspense>

        {/* Pricing Section */}
        <Suspense fallback={null}>
          <Pricing />
        </Suspense>

        {/* Performance Chart Section */}
        <section className="py-24 px-6 bg-background border-t border-dashed border-border/40">
          <div className="mx-auto max-w-6xl">
            <div className="flex flex-col items-center mb-16 text-center">
              <span className="inline-block px-2 py-0.5 mb-6 text-[9px] font-bold uppercase tracking-[0.25em] text-muted-foreground/60 border border-dashed border-border/60 rounded-sm bg-secondary/30">
                Benchmarks
              </span>
              <h2 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-4 text-foreground">
                Serious Performance.
              </h2>
              <p className="text-muted-foreground text-sm font-medium max-w-xl">
                We don't just guess. We benchmark every deploy. See how we stack up against industry standards in real-time latency tests.
              </p>
            </div>
            <Suspense fallback={<div className="h-[300px] bg-muted/30 rounded-xl animate-pulse" />}>
              <PerformanceChart />
            </Suspense>
          </div>
        </section>

        {/* Simple CTA */}
        <section className="py-24 px-6 border-t border-dashed border-border/40 bg-secondary/20">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-6 text-foreground">Build with integrity.</h2>
            <p className="text-muted-foreground mb-10 text-base font-medium">
              The simplest API for sending and receiving emails. No complexity, just email.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
              <LazySignInButton mode="modal">
                <Button className="h-9 px-8 rounded-md text-xs font-semibold w-full sm:w-auto">
                  Get Started Free
                </Button>
              </LazySignInButton>
              <Button asChild variant="outline" className="h-9 px-8 rounded-md text-xs font-semibold border-dashed border-border bg-background w-full sm:w-auto">
                <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer">
                  View Documentation
                </a>
              </Button>
            </div>
          </div>
        </section>

      </main>
    </>
  )
}
