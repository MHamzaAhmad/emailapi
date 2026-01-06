import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'

export const Route = createFileRoute('/admin/')({
    component: AdminIndexPage,
})

function AdminIndexPage() {
    const navigate = useNavigate()

    useEffect(() => {
        navigate({ to: '/admin/users' })
    }, [navigate])

    return (
        <div className="animate-pulse text-muted-foreground text-sm py-8 text-center">
            Redirecting to users...
        </div>
    )
}
