import { SignedIn, SignedOut, RedirectToSignIn } from '@clerk/clerk-react'

interface AuthGuardProps {
    children: React.ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
    return (
        <>
            <SignedIn>{children}</SignedIn>
            <SignedOut>
                <RedirectToSignIn />
            </SignedOut>
        </>
    )
}

interface PublicOnlyProps {
    children: React.ReactNode
    fallback?: React.ReactNode
}

export function PublicOnly({ children, fallback }: PublicOnlyProps) {
    return (
        <>
            <SignedOut>{children}</SignedOut>
            <SignedIn>{fallback}</SignedIn>
        </>
    )
}
