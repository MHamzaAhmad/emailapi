import { createFileRoute, Link } from '@tanstack/react-router'
import { SignUp } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import { ArrowRight01Icon } from '@hugeicons/core-free-icons'

export const Route = createFileRoute('/sign-up')(
    {
        component: SignUpPage,
    }
)

function SignUpPage() {
    return (
        <div className="min-h-screen bg-background flex flex-col">
            {/* Header */}
            <header className="border-b border-border">
                <div className="mx-auto max-w-5xl flex items-center justify-between px-6 h-12">
                    <Link to="/" className="flex items-center gap-1.5">
                        <div className="h-5 w-5 rounded border border-foreground/30 flex items-center justify-center">
                            <HugeiconsIcon icon={ArrowRight01Icon} size={12} strokeWidth={1.5} />
                        </div>
                        <span className="text-sm font-medium tracking-tight">emailapi</span>
                    </Link>
                </div>
            </header>

            {/* Sign Up Form */}
            <main className="flex-1 flex items-center justify-center p-6">
                <SignUp
                    routing="path"
                    path="/sign-up"
                    signInUrl="/sign-in"
                    forceRedirectUrl="/dashboard"
                />
            </main>
        </div>
    )
}
