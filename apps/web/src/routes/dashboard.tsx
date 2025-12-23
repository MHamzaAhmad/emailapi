import { createFileRoute, Link } from '@tanstack/react-router'
import { useUser } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    PlusSignIcon,
    FilterHorizontalIcon,
    Search01Icon,
    MoreHorizontalIcon,
    CheckmarkCircle01Icon,
    AlertCircleIcon,
    Clock01Icon,
    ArrowRight01Icon
} from '@hugeicons/core-free-icons'
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DataTable } from '@/components/ui/data-table'
import { ColumnDef } from "@tanstack/react-table"
import { Switch } from "@/components/ui/switch"

export const Route = createFileRoute('/dashboard')(
    {
        component: DashboardPage,
    }
)

function DashboardPage() {
    return (
        <AuthGuard>
            <DashboardContent />
        </AuthGuard>
    )
}

// Define Event Type
type Event = {
    id: string
    type: string
    target: string
    status: 'success' | 'failed' | 'pending'
    latency: string
    created_at: string
}

const data: Event[] = [
    { id: 'evt_8f2a1b', type: 'Email Dispatch', target: 'user@example.com', status: 'success', latency: '124ms', created_at: '2 mins ago' },
    { id: 'evt_9k3b2c', type: 'Email Dispatch', target: 'hello@company.com', status: 'success', latency: '156ms', created_at: '5 mins ago' },
    { id: 'evt_2p4l5d', type: 'Webhook', target: 'https://api.acme.inc/wh', status: 'failed', latency: '42ms', created_at: '12 mins ago' },
    { id: 'evt_7m9x0e', type: 'Health Check', target: 'us-east-cluster', status: 'success', latency: '12ms', created_at: '15 mins ago' },
    { id: 'evt_1z8q9f', type: 'Email Dispatch', target: 'billing@client.io', status: 'pending', latency: '-', created_at: 'Just now' },
    { id: 'evt_3w5r7g', type: 'API Key Created', target: 'System', status: 'success', latency: '24ms', created_at: '1 hour ago' },
]

const columns: ColumnDef<Event>[] = [
    {
        accessorKey: "id",
        header: "Event ID",
        cell: ({ row }) => <span className="font-mono text-xs text-muted-foreground">{row.getValue("id")}</span>,
    },
    {
        accessorKey: "type",
        header: "Type",
        cell: ({ row }) => <span className="font-medium text-xs">{row.getValue("type")}</span>,
    },
    {
        accessorKey: "target",
        header: "Target / Recipient",
        cell: ({ row }) => <span className="text-xs text-muted-foreground font-mono">{row.getValue("target")}</span>,
    },
    {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => {
            const status = row.getValue("status") as string
            return (
                <div className="flex items-center gap-2">
                    <Switch checked={status === 'success'} className="scale-75 data-[state=checked]:bg-emerald-500" />
                    <span className={`text-[10px] font-medium uppercase tracking-wide ${status === 'success' ? 'text-emerald-600' :
                        status === 'failed' ? 'text-red-500' : 'text-amber-500'
                        }`}>
                        {status}
                    </span>
                </div>
            )
        },
    },
    {
        accessorKey: "latency",
        header: "Latency",
        cell: ({ row }) => <span className="font-mono text-xs text-muted-foreground">{row.getValue("latency")}</span>,
    },
    {
        accessorKey: "created_at",
        header: "Time",
        cell: ({ row }) => <span className="text-xs text-muted-foreground">{row.getValue("created_at")}</span>,
    },
    {
        id: "actions",
        cell: ({ row }) => {
            return (
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button variant="ghost" className="h-8 w-8 p-0">
                            <span className="sr-only">Open menu</span>
                            <HugeiconsIcon icon={MoreHorizontalIcon} size={16} />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem>View Details</DropdownMenuItem>
                        <DropdownMenuItem>Replay Event</DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            )
        },
    },
]

function DashboardContent() {
    const { user } = useUser()

    return (
        <Shell>
            {/* Header Area */}
            <div className="flex flex-col gap-6 mb-8">
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div className="flex items-center gap-2 text-2xl font-bold tracking-tight">
                        <h1>Dashboard</h1>
                        <span className="text-muted-foreground font-light">/</span>
                        <h1 className="text-foreground">Activity</h1>
                    </div>
                    <div className="flex items-center gap-2">
                        <Button variant="outline" size="sm" className="h-9 gap-2 bg-background hover:bg-muted/50 border-input/60 shadow-sm">
                            <HugeiconsIcon icon={FilterHorizontalIcon} size={14} />
                            <span>Filters</span>
                        </Button>
                        <Button size="sm" className="h-9 gap-2 shadow-sm font-medium">
                            <HugeiconsIcon icon={PlusSignIcon} size={14} />
                            <span>Dispatch Email</span>
                        </Button>
                    </div>
                </div>


            </div>

            {/* Main Content - Data Table */}
            <div className="space-y-4">
                <DataTable columns={columns} data={data} searchKey="target" />
            </div>

            <div className="mt-4 text-xs text-muted-foreground">
                Showing {data.length} of {data.length} events
            </div>
        </Shell>
    )
}
