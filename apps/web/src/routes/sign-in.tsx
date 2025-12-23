import { createFileRoute, Link } from '@tanstack/react-router'
import { SignIn } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import { ArrowRight01Icon } from '@hugeicons/core-free-icons'

export const Route = createFileRoute('/sign-in')(
    {
        component: SignInPage,
    }
)

function SignInPage() {
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

            {/* Sign In Form */}
            <main className="flex-1 flex items-center justify-center p-6">
                <SignIn
                    routing="path"
                    path="/sign-in"
                    signUpUrl="/sign-up"
                    forceRedirectUrl="/dashboard"
                />
            </main>
        </div>
    )
}
