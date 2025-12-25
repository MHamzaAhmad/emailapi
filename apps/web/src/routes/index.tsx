import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { SignedIn, SignedOut, SignInButton, UserButton } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  PackageIcon,
  ZapIcon,
  Mail01Icon,
  LockKeyIcon,
  Book02Icon
} from '@hugeicons/core-free-icons'
import { useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { CodeWindow } from '@/components/ui/code-window'
import { ModeToggle } from '@/components/mode-toggle'
import { Features } from '@/components/landing/Features'

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
    <div className="min-h-screen bg-background text-foreground font-sans flex flex-col">
      {/* Navigation - Matching Shell Header Exactly */}
      <nav className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 md:px-6">
          <div className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:text-foreground transition-colors">
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
              <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
            </div>
            <span className="text-sm">emailapi.dev</span>
          </div>
          <div className="hidden md:flex items-center gap-4">
            <ModeToggle />
            {/* @ts-expect-error docs route */}
            <Link to="/docs" className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">Documentation</Link>
            <SignedOut>
              <SignInButton mode="modal">
                <Button size="sm" className="h-8 px-4 text-xs font-medium">
                  Login
                </Button>
              </SignInButton>
            </SignedOut>
            <SignedIn>
              <UserButton />
            </SignedIn>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <main className="flex-1">
        <section className="relative px-6 pt-24 pb-12 overflow-hidden">
          <div className="mx-auto max-w-4xl text-center">
            <h1 className="mb-6 text-4xl font-bold tracking-tight sm:text-6xl lg:text-7xl leading-[1.1]">
              The Simplest API for <br /> <span className="text-primary">Inbound & Outbound</span>
            </h1>
            <p className="mx-auto mb-10 text-base sm:text-lg text-muted-foreground leading-relaxed max-w-2xl">
              High-performance email infrastructure without the complexity. <br className="hidden sm:block" />
              Send transactional emails or build powerful reply flows in minutes.
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-3 mb-20">
              <Button asChild className="h-9 px-6 text-sm shadow-sm rounded-md">
                {/* @ts-expect-error auth route */}
                <Link to="/sign-up">Start Building</Link>
              </Button>
              <Button asChild variant="outline" className="h-9 px-6 text-sm bg-background hover:bg-muted/50 rounded-md">
                {/* @ts-expect-error docs route */}
                <Link to="/docs">
                  <HugeiconsIcon icon={Book02Icon} size={14} className="mr-2 text-muted-foreground" />
                  Documentation
                </Link>
              </Button>
            </div>

            {/* Centered Code Window */}
            <div className="relative mx-auto max-w-3xl">
              {/* Subtle Back Glow */}
              <div className="absolute -inset-4 rounded-[2rem] bg-gradient-to-b from-primary/5 to-transparent blur-3xl opacity-50" />

              <CodeWindow
                className="w-full border-border/60 shadow-xl"
                tabs={[
                  {
                    label: "TypeScript",
                    value: "ts",
                    language: "typescript",
                    content: `import { createClient } from 'emailapi-sdk'

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
                    content: `curl -X POST https://api.emailapi.dev/v1/send \\\n  -H "Authorization: Bearer em_live_..." \\\n  -d '{\n    "from": "updates@app.com",\n    "to": ["user@example.com"],\n    "subject": "Welcome!",\n    "body": "Thanks for signing up."\n  }'`
                  },
                  {
                    label: "Go",
                    value: "go",
                    language: "go",
                    content: `package main\n\nimport "github.com/emailapi/go-sdk"\n\nfunc main() {\n  client := emailapi.NewClient("em_live_...")\n\n  // Send Email\n  client.Send(&emailapi.Message{\n    From:    "updates@app.com",\n    To:      []string{"user@example.com"},\n    Subject: "Welcome!",\n    Body:    "Thanks for signing up.",\n  })\n}`
                  },
                  {
                    label: "gRPC",
                    value: "grpc",
                    language: "bash",
                    content: `grpcurl -d '{\n  "from": "updates@app.com",\n  "to": ["user@example.com"],\n  "subject": "Welcome!",\n  "body": "Thanks for signing up."\n}' \\\n  -H "Authorization: Bearer em_live_..." \\\n  api.emailapi.dev:443 emailapi.v1.EmailService/Send`
                  }
                ]}
              />
            </div>
          </div>

          {/* Background Grid */}
          <div className="absolute inset-0 -z-10 h-full w-full bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:32px_32px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]" />
        </section>

        {/* Bento Grid Features Section */}
        <Features />

        {/* Simple CTA */}
        <section className="py-24 px-6 border-t border-dashed border-border/40">
          <div className="mx-auto max-w-xl text-center">
            <h2 className="text-2xl font-bold tracking-tight mb-4">Ready to Ship?</h2>
            <p className="text-muted-foreground mb-8 text-sm">
              Get your API key in seconds. 5,000 free emails per month.
            </p>
            <Button asChild className="h-9 px-8 rounded-md text-sm">
              {/* @ts-expect-error auth route */}
              <Link to="/sign-up">Start Building Now</Link>
            </Button>
          </div>
        </section>

      </main>

      {/* Minimal Footer */}
      <footer className="border-t py-12 px-6 bg-background">
        <div className="mx-auto max-w-5xl flex flex-col md:flex-row justify-between items-center gap-6">
          <div className="flex items-center gap-2 grayscale opacity-50 hover:grayscale-0 hover:opacity-100 transition-all">
            <div className="flex h-5 w-5 items-center justify-center rounded-sm bg-foreground text-background">
              <HugeiconsIcon icon={PackageIcon} size={12} strokeWidth={3} />
            </div>
            <span className="text-xs font-bold tracking-widest text-foreground uppercase">emailapi</span>
          </div>
          <div className="flex gap-6 text-xs text-muted-foreground">
            {/* @ts-expect-error docs route */}
            <Link to="/docs" className="hover:text-foreground">Docs</Link>
            {/* @ts-expect-error status route */}
            <Link to="/status" className="hover:text-foreground">Status</Link>
            {/* @ts-expect-error legal route */}
            <Link to="/legal" className="hover:text-foreground">Legal</Link>
          </div>
        </div>
      </footer>
    </div>
  )
}
