import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { SignedIn, SignedOut, SignInButton, Waitlist, UserButton } from '@clerk/clerk-react'
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
import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"

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
            <span className="text-sm">SimpleEmailAPI</span>
          </div>
          <div className="hidden md:flex items-center gap-4">
            <ModeToggle />
            <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">Documentation</a>
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
              High-performance email infrastructure for modern teams. <br className="hidden sm:block" />
              Currently in private beta. Login if you have an invite.
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-3 mb-20">
              <Dialog>
                <DialogTrigger asChild>
                  <Button className="h-9 px-6 text-sm shadow-sm rounded-md">
                    Join Waitlist
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-fit border-none bg-transparent p-0 shadow-none">
                  <Waitlist />
                </DialogContent>
              </Dialog>
              <Button asChild variant="outline" className="h-9 px-6 text-sm bg-background hover:bg-muted/50 rounded-md">
                <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer">
                  <HugeiconsIcon icon={Book02Icon} size={14} className="mr-2 text-muted-foreground" />
                  Documentation
                </a>
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
                    content: `curl -X POST https://api.simpleemailapi.dev/v1/send \\\n  -H "Authorization: Bearer em_live_..." \\\n  -d '{\n    "from": "updates@app.com",\n    "to": ["user@example.com"],\n    "subject": "Welcome!",\n    "body": "Thanks for signing up."\n  }'`
                  },
                  {
                    label: "Go",
                    value: "go",
                    language: "go",
                    content: `package main\n\nimport "github.com/simpleemailapi/go-sdk"\n\nfunc main() {\n  client := simpleemailapi.NewClient("em_live_...")\n\n  // Send Email\n  client.Send(&simpleemailapi.Message{\n    From:    "updates@app.com",\n    To:      []string{"user@example.com"},\n    Subject: "Welcome!",\n    Body:    "Thanks for signing up.",\n  })\n}`
                  },
                  {
                    label: "gRPC",
                    value: "grpc",
                    language: "bash",
                    content: `grpcurl -d '{\n  "from": "updates@app.com",\n  "to": ["user@example.com"],\n  "subject": "Welcome!",\n  "body": "Thanks for signing up."\n}' \\\n  -H "Authorization: Bearer em_live_..." \\\n  api.simpleemailapi.dev:443 simpleemailapi.v1.EmailService/Send`
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
            <h2 className="text-2xl font-bold tracking-tight mb-4">Join the Private Beta</h2>
            <p className="text-muted-foreground mb-8 text-sm">
              We are currently in invite-only mode. If you have an invite code, login to get started.
            </p>
            <SignInButton mode="modal">
              <Button className="h-9 px-8 rounded-md text-sm">
                Login
              </Button>
            </SignInButton>
          </div>
        </section>

      </main>

      {/* Value Driven Footer */}
      <footer className="relative border-t bg-background overflow-hidden">
        {/* Huge BG Text */}
        <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full select-none overflow-hidden pointer-events-none">
          <h1 className="text-[15rem] md:text-[20rem] font-bold text-foreground/5 whitespace-nowrap text-center tracking-tighter leading-none">
            EMAILS
          </h1>
        </div>

        <div className="relative z-10 mx-auto max-w-7xl px-6 py-20">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-12">
            <div className="col-span-1 md:col-span-2 space-y-6">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
                  <HugeiconsIcon icon={PackageIcon} size={18} strokeWidth={2.5} />
                </div>
                <span className="text-lg font-bold tracking-tight">SimpleEmailAPI</span>
              </div>
              <p className="text-muted-foreground max-w-xs leading-relaxed">
                The easiest way to send and receive emails. Built for developers.
              </p>
            </div>

            <div className="md:col-start-4 space-y-4">
              <h4 className="text-sm font-semibold tracking-wider uppercase text-foreground">Product</h4>
              <ul className="space-y-3 text-sm text-muted-foreground">
                <li>
                  <a href="https://docs.simpleemailapi.dev" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Documentation</a>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-20 pt-8 border-t border-border/40 flex flex-col md:flex-row justify-between items-center gap-4 text-xs text-muted-foreground">
            <p>&copy; {new Date().getFullYear()} SimpleEmailAPI. All rights reserved.</p>
            <div className="flex gap-2 items-center">
              <div className="h-2 w-2 rounded-full bg-green-500 animate-pulse"></div>
              <span className="font-mono">All systems operational</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
