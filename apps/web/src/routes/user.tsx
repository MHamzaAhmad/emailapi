import { createFileRoute, useNavigate } from '@tanstack/react-router'
import {
    User as UserIcon,
    Mail,
    Shield,
    CheckCircle,
    AlertCircle,
    Loader2,
    LogOut,
    Key,
    Terminal,
    Copy,
} from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useCreateUser, useCurrentUser } from '@/hooks'
import type { CreateUserRequest } from '@/types'
import { setAuthToken, getAuthToken, clearAuthToken } from '@/lib/api'

export const Route = createFileRoute('/user')({
    component: UserTestPage,
})

function UserTestPage() {
    const [token, setToken] = useState(getAuthToken())
    const navigate = useNavigate()
    const { data: user, isLoading: isLoadingUser, refetch } = useCurrentUser()

    const handleLogout = () => {
        clearAuthToken()
        setToken(null)
        navigate({ to: '/user' })
    }

    const handleLoginSuccess = () => {
        setToken(getAuthToken())
        refetch()
    }

    if (isLoadingUser && token) {
        return (
            <div className="min-h-screen bg-slate-950 flex items-center justify-center">
                <Loader2 className="w-8 h-8 text-cyan-500 animate-spin" />
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-slate-950 p-6 md:p-12 font-sans text-slate-200">
            <div className="max-w-2xl mx-auto space-y-8">

                {/* Header */}
                <div className="text-center space-y-2">
                    <h1 className="text-3xl font-bold text-white tracking-tight">
                        Test Session Manager
                    </h1>
                    <p className="text-slate-400">
                        Create a temporary user to test the Email API
                    </p>
                </div>

                {!token ? (
                    <CreateUserForm onSuccess={handleLoginSuccess} />
                ) : (
                    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
                        {/* User Profile Card */}
                        <div className="bg-slate-900/50 border border-slate-800 rounded-2xl p-6 backdrop-blur-sm">
                            <div className="flex items-center justify-between mb-6">
                                <div className="flex items-center gap-3">
                                    <div className="w-12 h-12 rounded-full bg-cyan-500/10 flex items-center justify-center text-cyan-400">
                                        <UserIcon className="w-6 h-6" />
                                    </div>
                                    <div>
                                        <h2 className="text-xl font-semibold text-white">Active Session</h2>
                                        <p className="text-sm text-cyan-400 flex items-center gap-1.5">
                                            <span className="w-2 h-2 rounded-full bg-cyan-500 animate-pulse" />
                                            Connected
                                        </p>
                                    </div>
                                </div>
                                <button
                                    onClick={handleLogout}
                                    className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                                >
                                    <LogOut className="w-4 h-4" />
                                    End Session
                                </button>
                            </div>

                            {user && (
                                <div className="grid gap-4 p-4 bg-slate-950/50 rounded-xl border border-slate-800/50">
                                    <div className="flex items-center justify-between group">
                                        <div className="flex items-center gap-3 text-slate-400">
                                            <UserIcon className="w-4 h-4" />
                                            <span className="text-sm font-medium">Name</span>
                                        </div>
                                        <span className="text-slate-200 font-medium font-mono">{user.name}</span>
                                    </div>
                                    <div className="h-px bg-slate-800/50" />
                                    <div className="flex items-center justify-between group">
                                        <div className="flex items-center gap-3 text-slate-400">
                                            <Mail className="w-4 h-4" />
                                            <span className="text-sm font-medium">Email</span>
                                        </div>
                                        <span className="text-slate-200 font-medium font-mono">{user.email}</span>
                                    </div>
                                    <div className="h-px bg-slate-800/50" />
                                    <div className="flex items-center justify-between group">
                                        <div className="flex items-center gap-3 text-slate-400">
                                            <Shield className="w-4 h-4" />
                                            <span className="text-sm font-medium">Role</span>
                                        </div>
                                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 capitalize">
                                            {user.role}
                                        </span>
                                    </div>
                                </div>
                            )}
                        </div>

                        {/* API Key Section */}
                        <div className="bg-slate-900/50 border border-slate-800 rounded-2xl p-6 backdrop-blur-sm">
                            <div className="flex items-start gap-4 mb-4">
                                <div className="p-2 bg-yellow-500/10 rounded-lg text-yellow-500">
                                    <Key className="w-5 h-5" />
                                </div>
                                <div>
                                    <h3 className="text-lg font-semibold text-white">Authentication Token</h3>
                                    <p className="text-slate-400 text-sm mt-1">
                                        This token is automatically saved in your browser's local storage and used for all API requests.
                                    </p>
                                </div>
                            </div>

                            <div className="bg-slate-950 rounded-lg border border-slate-800 p-4 relative group">
                                <code className="text-sm text-slate-300 font-mono break-all pr-12 block">
                                    {token}
                                </code>
                                <button
                                    onClick={() => navigator.clipboard.writeText(token || '')}
                                    className="absolute top-3 right-3 p-2 bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white rounded-md transition-colors"
                                    title="Copy Token"
                                >
                                    <Copy className="w-4 h-4" />
                                </button>
                            </div>
                        </div>

                        {/* Quick Actions */}
                        <div className="grid grid-cols-2 gap-4">
                            <button
                                onClick={() => navigate({ to: '/domains' })}
                                className="p-4 bg-slate-800/50 hover:bg-slate-800 border border-slate-700 hover:border-cyan-500/50 rounded-xl transition-all group text-left"
                            >
                                <div className="mb-2 p-2 w-fit rounded-lg bg-cyan-500/10 text-cyan-400 group-hover:bg-cyan-500 group-hover:text-white transition-colors">
                                    <Terminal className="w-5 h-5" />
                                </div>
                                <h3 className="font-semibold text-slate-200">Test Domains</h3>
                                <p className="text-sm text-slate-500 mt-1">Verify and manage sending domains</p>
                            </button>

                            <button
                                onClick={() => navigate({ to: '/api-keys' })}
                                className="p-4 bg-slate-800/50 hover:bg-slate-800 border border-slate-700 hover:border-purple-500/50 rounded-xl transition-all group text-left"
                            >
                                <div className="mb-2 p-2 w-fit rounded-lg bg-purple-500/10 text-purple-400 group-hover:bg-purple-500 group-hover:text-white transition-colors">
                                    <Key className="w-5 h-5" />
                                </div>
                                <h3 className="font-semibold text-slate-200">Manage Keys</h3>
                                <p className="text-sm text-slate-500 mt-1">Create additional API keys</p>
                            </button>
                        </div>

                    </div>
                )}
            </div>
        </div>
    )
}

function CreateUserForm({ onSuccess }: { onSuccess: () => void }) {
    const createUser = useCreateUser()
    const [formData, setFormData] = useState<CreateUserRequest>({
        name: '',
        email: '',
        role: 'admin', // Default to admin for testing power
    })
    const [isSubmitting, setIsSubmitting] = useState(false)

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault()
        setIsSubmitting(true)
        try {
            // 1. Create User (returns API key now)
            const response = await createUser.mutateAsync(formData)

            if (response.apiKey) {
                // 2. Set Token
                setAuthToken(response.apiKey)

                // 3. Update UI
                onSuccess()
            } else {
                console.error('No API key returned from user creation')
                setIsSubmitting(false)
            }

        } catch (err) {
            console.error('Failed to create test session:', err)
            setIsSubmitting(false)
        }
    }

    return (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 md:p-8 shadow-2xl">
            <div className="mb-6">
                <h2 className="text-xl font-bold text-white mb-2">Create Test User</h2>
                <p className="text-slate-400 text-sm">
                    This will create a new user and automatically generate an API key for this session.
                </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                    <label className="block text-slate-400 text-sm font-medium mb-1.5">
                        Full Name
                    </label>
                    <input
                        type="text"
                        required
                        value={formData.name}
                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        className="w-full px-4 py-2.5 bg-slate-950 border border-slate-700 rounded-lg text-white focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 focus:outline-none transition-all"
                        placeholder="Jane Tester"
                    />
                </div>

                <div>
                    <label className="block text-slate-400 text-sm font-medium mb-1.5">
                        Email Address
                    </label>
                    <input
                        type="email"
                        required
                        value={formData.email}
                        onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                        className="w-full px-4 py-2.5 bg-slate-950 border border-slate-700 rounded-lg text-white focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 focus:outline-none transition-all"
                        placeholder="jane@example.com"
                    />
                </div>

                <button
                    type="submit"
                    disabled={isSubmitting}
                    className="w-full py-3 bg-cyan-600 hover:bg-cyan-500 text-white font-semibold rounded-lg transition-all shadow-lg shadow-cyan-900/20 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 mt-2"
                >
                    {isSubmitting ? (
                        <>
                            <Loader2 className="w-5 h-5 animate-spin" />
                            Setting up session...
                        </>
                    ) : (
                        <>
                            Start Test Session
                            <CheckCircle className="w-5 h-5" />
                        </>
                    )}
                </button>
            </form>

            <div className="mt-6 p-4 bg-slate-950 rounded-lg border border-slate-800 flex gap-3">
                <AlertCircle className="w-5 h-5 text-indigo-400 shrink-0 mt-0.5" />
                <p className="text-xs text-slate-400">
                    <strong>Tip:</strong> The API key generated will have full <code>admin</code> privileges for testing all endpoints.
                </p>
            </div>
        </div>
    )
}
