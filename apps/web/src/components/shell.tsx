import { Sidebar } from './sidebar'

interface ShellProps {
    children: React.ReactNode
}

export function Shell({ children }: ShellProps) {
    return (
        <div className="min-h-screen bg-background">
            <Sidebar />
            <main className="pl-[200px]">
                <div className="mx-auto max-w-5xl p-6">
                    {children}
                </div>
            </main>
        </div>
    )
}

interface PageHeaderProps {
    title: string
    description?: string
    actions?: React.ReactNode
}

export function PageHeader({ title, description, actions }: PageHeaderProps) {
    return (
        <div className="flex items-start justify-between mb-6">
            <div>
                <h1 className="text-lg font-medium tracking-tight">{title}</h1>
                {description && (
                    <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
                )}
            </div>
            {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
    )
}
