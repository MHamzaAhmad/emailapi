'use client'

import { lazy, Suspense, ComponentProps, ReactNode } from 'react'

// Lazy-loaded Clerk components for performance
// These are only loaded when actually rendered, reducing initial bundle size

const ClerkSignInButton = lazy(() =>
    import('@clerk/clerk-react').then(m => ({ default: m.SignInButton }))
)

const ClerkSignedIn = lazy(() =>
    import('@clerk/clerk-react').then(m => ({ default: m.SignedIn }))
)

const ClerkSignedOut = lazy(() =>
    import('@clerk/clerk-react').then(m => ({ default: m.SignedOut }))
)

const ClerkUserButton = lazy(() =>
    import('@clerk/clerk-react').then(m => ({ default: m.UserButton }))
)

// Wrapper components with Suspense

type SignInButtonProps = ComponentProps<typeof ClerkSignInButton>
export function LazySignInButton(props: SignInButtonProps) {
    return (
        <Suspense fallback={null}>
            <ClerkSignInButton {...props} />
        </Suspense>
    )
}

type SignedInProps = { children: ReactNode }
export function LazySignedIn({ children }: SignedInProps) {
    return (
        <Suspense fallback={null}>
            <ClerkSignedIn>{children}</ClerkSignedIn>
        </Suspense>
    )
}

type SignedOutProps = { children: ReactNode }
export function LazySignedOut({ children }: SignedOutProps) {
    return (
        <Suspense fallback={null}>
            <ClerkSignedOut>{children}</ClerkSignedOut>
        </Suspense>
    )
}

type UserButtonProps = ComponentProps<typeof ClerkUserButton>
export function LazyUserButton(props: UserButtonProps) {
    return (
        <Suspense fallback={<div className="h-7 w-7 rounded-full bg-muted animate-pulse" />}>
            <ClerkUserButton {...props} />
        </Suspense>
    )
}
