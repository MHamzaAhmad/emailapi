import { SignedIn, SignedOut, RedirectToSignIn, useAuth } from '@clerk/clerk-react'
import { useEffect } from 'react'
import { setTokenGetter } from '@/lib/api'

interface AuthGuardProps {
    children: React.ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
    const { getToken } = useAuth()

    useEffect(() => {
        setTokenGetter(getToken)
    }, [getToken])

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
