import { createFileRoute, useNavigate, Link } from '@tanstack/react-router'
import { SignedIn, SignedOut, SignInButton, Waitlist, UserButton } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  PackageIcon,
  Menu01Icon
} from '@hugeicons/core-free-icons'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { CodeWindow } from '@/components/ui/code-window'
import { ModeToggle } from '@/components/mode-toggle'
import { GridOfTruth } from '@/components/landing/GridOfTruth'
import { PerformanceChart } from '@/components/landing/PerformanceChart'
import { ComparisonSection } from '@/components/landing/ComparisonSection'
import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Sheet,
  SheetContent,
  SheetTrigger,
} from "@/components/ui/sheet"

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
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <div className="min-h-screen bg-background text-foreground font-sans flex flex-col antialiased overflow-x-hidden w-full">
      {/* Navigation - Aligned with Dashboard Shell */}
      <nav className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 md:px-6">
          <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/90">
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
              <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
            </div>
            <span className="text-sm">SimpleEmailAPI</span>
          </div>

          {/* Desktop Nav */}
          <div className="hidden md:flex items-center gap-4">
            <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">Documentation</a>
            <SignedOut>
              <SignInButton mode="modal">
                <Button size="sm" variant="outline" className="h-8 px-4 text-xs font-medium border-dashed border-border hover:bg-muted/50 rounded-md">
                  Login
                </Button>
              </SignInButton>
            </SignedOut>
            <SignedIn>
              <UserButton
                appearance={{
                  elements: {
                    avatarBox: "h-7 w-7 rounded-full ring-2 ring-background hover:ring-muted transition-all"
                  }
                }}
              />
            </SignedIn>
            <div className="h-4 w-[1px] bg-border/60 mx-1" />
            <ModeToggle />
          </div>

          {/* Mobile Menu Trigger */}
          <div className="md:hidden flex items-center gap-2">
            <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
              <SheetTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <HugeiconsIcon icon={Menu01Icon} size={18} />
                </Button>
              </SheetTrigger>
              <SheetContent side="right" className="w-[300px] border-l border-border/40 bg-background/95 backdrop-blur-xl p-6">
                <div className="flex flex-col gap-6 mt-6">
                  <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 mb-4">
                    <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                      <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
                    </div>
                    <span className="text-sm">SimpleEmailAPI</span>
                  </div>

                  <div className="flex flex-col gap-4">
                    <a
                      href="https://docs.simpleemailapi.dev"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors border-b border-dashed border-border/40 pb-2"
                    >
                      Documentation
                    </a>

                    <div className="flex flex-col gap-3 pt-2">
                      <Dialog>
                        <DialogTrigger asChild>
                          <Button className="w-full h-9 rounded-md text-xs font-semibold justify-start" variant="outline">
                            Join Waitlist
                          </Button>
                        </DialogTrigger>
                        <DialogContent className="max-w-fit border-none bg-transparent p-0 shadow-none">
                          <Waitlist />
                        </DialogContent>
                      </Dialog>

                      <SignedOut>
                        <SignInButton mode="modal">
                          <Button className="w-full h-9 rounded-md text-xs font-semibold justify-start">
                            Login
                          </Button>
                        </SignInButton>
                      </SignedOut>
                    </div>
                  </div>

                  <div className="mt-auto pt-6 border-t border-dashed border-border/40 flex items-center justify-between">
                    <span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">Theme</span>
                    <ModeToggle />
                  </div>
                </div>
              </SheetContent>
            </Sheet>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <main className="flex-1 w-full overflow-hidden">
        <section className="relative px-4 sm:px-6 pt-32 sm:pt-24 pb-16 sm:pb-20 overflow-hidden border-b border-dashed border-border/40">
          <div className="mx-auto max-w-4xl text-center">
            <h1 className="mb-6 text-4xl sm:text-5xl md:text-7xl font-bold tracking-tighter leading-[1.1] text-foreground">
              Simple. Performant. <br /> <span className="text-muted-foreground/40 font-medium">Privacy Focused.</span>
            </h1>
            <p className="mx-auto mb-10 text-sm sm:text-base text-muted-foreground leading-relaxed max-w-xl font-medium px-4">
              Enterprise-grade email infrastructure for modern engineering teams. <br className="hidden sm:block" />
              Pay only for what you use. <span className="text-foreground">$0.25 per 1,000 emails</span>.
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-3 mb-16 sm:mb-24 px-4 w-full">
              <Dialog>
                <DialogTrigger asChild>
                  <Button className="h-9 px-6 text-xs font-semibold shadow-sm rounded-md w-full sm:w-auto">
                    Join Waitlist
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-fit border-none bg-transparent p-0 shadow-none">
                  <Waitlist />
                </DialogContent>
              </Dialog>
              <Button asChild variant="outline" className="h-9 px-6 text-xs font-semibold bg-background hover:bg-muted/50 rounded-md border-dashed border-border w-full sm:w-auto">
                <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer">
                  API Documentation
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
                    content: `import { createClient } from 'simpleemailapi-sdk'

const client = createClient({ apiKey: 'em_...' })

// Send an email
await client.send({
  from: 'hello@yourapp.com',
  to: ['user@example.com'],
  subject: 'Welcome!',
  body: 'Thanks for signing up.'
})

// Listen for replies
client.onReceive({
  onReplied: (reply) => console.log('New reply:', reply.body)
})`
                  },
                  {
                    label: "cURL",
                    value: "curl",
                    language: "bash",
                    content: `curl -X POST https://api.simpleemailapi.dev/v1/send \\
  -H "Authorization: Bearer sea_live_..." \\
  -d '{
    "from": "updates@app.com",
    "to": ["user@example.com"],
    "subject": "Welcome!",
    "body": "Thanks for signing up."
  }'`
                  },
                  {
                    label: "Go",
                    value: "go",
                    language: "go",
                    content: `package main

import "github.com/simpleemailapi/go-sdk"

func main() {
  client := simpleemailapi.NewClient("sea_live_...")

  // Send Email
  client.Send(&simpleemailapi.Message{
    From:    "updates@app.com",
    To:      []string{"user@example.com"},
    Subject: "Welcome!",
    Body:    "Thanks for signing up.",
  })
}`
                  }
                ]}
              />
            </div>
          </div>

          {/* Background Grid - Subtle Dashed */}
          <div className="absolute inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />
        </section>

        {/* Symmetric Grid of Truth */}
        <GridOfTruth />

        {/* Comparison Section */}
        <ComparisonSection />

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
            <PerformanceChart />
          </div>
        </section>

        {/* Simple CTA */}
        <section className="py-24 px-6 border-t border-dashed border-border/40 bg-secondary/20">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl sm:text-4xl font-bold tracking-tighter mb-6 text-foreground">Build with integrity.</h2>
            <p className="text-muted-foreground mb-10 text-base font-medium">
              Join the private beta and experience the next generation of email infrastructure.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
              <SignInButton mode="modal">
                <Button className="h-9 px-8 rounded-md text-xs font-semibold w-full sm:w-auto">
                  Login to Console
                </Button>
              </SignInButton>
              <Dialog>
                <DialogTrigger asChild>
                  <Button variant="outline" className="h-9 px-8 rounded-md text-xs font-semibold border-dashed border-border bg-background w-full sm:w-auto">
                    Join Waitlist
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-fit border-none bg-transparent p-0 shadow-none">
                  <Waitlist />
                </DialogContent>
              </Dialog>
            </div>
          </div>
        </section>

      </main>

      {/* Minimalist Footer */}
      <footer className="border-t border-dashed border-border/40 bg-background py-14 px-6">
        <div className="mx-auto max-w-7xl flex flex-col md:flex-row justify-between items-center gap-8">
          <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/80">
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-secondary text-foreground shadow-sm ring-1 ring-black/5 dark:ring-white/10">
              <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
            </div>
            <span className="text-xs">SimpleEmailAPI</span>
          </div>

          <div className="flex items-center gap-6 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
            <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Documentation</a>
            <span className="text-border/40">|</span>
            <Link to="/terms" className="hover:text-foreground transition-colors">Terms</Link>
            <span className="text-border/40">|</span>
            <Link to="/privacy" className="hover:text-foreground transition-colors">Privacy</Link>
            <span className="text-border/40">|</span>
            <p>&copy; {new Date().getFullYear()} SimpleEmailAPI</p>
          </div>

          <div className="flex gap-2 items-center px-3 py-1 rounded-md border border-dashed border-border/60 bg-secondary/30">
            <div className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse ring-2 ring-emerald-500/20"></div>
            <span className="text-[9px] font-bold uppercase tracking-[0.2em] text-muted-foreground">Systems Nominal</span>
          </div>
        </div>
      </footer>
    </div>
  )
}
