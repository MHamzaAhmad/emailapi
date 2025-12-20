import { createFileRoute } from '@tanstack/react-router'
import {
    User as UserIcon,
    Mail,
    Shield,
    Plus,
    Search,
    CheckCircle,
    AlertCircle,
    Loader2,
    X,
    Edit2,
    Clock,
} from 'lucide-react'
import { useState, type FormEvent, useEffect } from 'react'
import { useUsers, useCreateUser, useUpdateUser } from '@/hooks'
import { useCreateApiKey } from '@/hooks'
import type { CreateUserRequest, User, UserRole } from '@/types'
import { setAuthToken } from '@/lib/api'

export const Route = createFileRoute('/user')({
    component: UserManagementPage,
})

function UserModal({
    isOpen,
    onClose,
    user,
}: {
    isOpen: boolean
    onClose: () => void
    user?: User
}) {
    const createUser = useCreateUser()
    const updateUser = useUpdateUser()
    const createApiKey = useCreateApiKey()

    const [formData, setFormData] = useState<CreateUserRequest & { isActive?: boolean }>({
        name: '',
        email: '',
        role: 'member',
        isActive: true,
    })
    const [createdApiKey, setCreatedApiKey] = useState<string | null>(null)

    useEffect(() => {
        if (user) {
            setFormData({
                name: user.name,
                email: user.email,
                role: user.role,
                isActive: user.isActive,
            })
        } else {
            setFormData({
                name: '',
                email: '',
                role: 'member',
                isActive: true, // Default active for new users
            })
            setCreatedApiKey(null)
        }
    }, [user, isOpen])

    if (!isOpen) return null

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault()
        try {
            if (user) {
                await updateUser.mutateAsync({
                    id: user.id,
                    data: formData,
                })
                onClose()
            } else {
                // Create user
                const result = await createUser.mutateAsync(formData)

                // Auto-create API key for the new user
                const apiKeyResult = await createApiKey.mutateAsync({
                    name: `${formData.name}'s API Key`,
                    scopes: ['email:send', 'email:read', 'domain:read', 'domain:write', 'apikey:read', 'apikey:write', 'user:read', 'user:write'],
                    environment: 'live',
                })

                // Store API key in localStorage
                setAuthToken(apiKeyResult.rawKey)
                setCreatedApiKey(apiKeyResult.rawKey)
            }
        } catch (err) {
            console.error('Failed to save user:', err)
        }
    }

    const isPending = createUser.isPending || updateUser.isPending || createApiKey.isPending

    // If API key was just created, show it
    if (createdApiKey) {
        return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
                <div className="bg-slate-800 border border-slate-700 rounded-xl w-full max-w-md shadow-2xl overflow-hidden">
                    <div className="flex items-center justify-between p-6 border-b border-slate-700 bg-green-500/10">
                        <div className="flex items-center gap-3">
                            <CheckCircle className="w-6 h-6 text-green-400" />
                            <h2 className="text-xl font-bold text-white">User Created Successfully!</h2>
                        </div>
                    </div>

                    <div className="p-6 space-y-4">
                        <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                            <div className="flex items-start gap-3">
                                <AlertCircle className="w-5 h-5 text-yellow-400 mt-0.5" />
                                <div>
                                    <p className="text-yellow-400 font-semibold mb-2">Save this API key!</p>
                                    <p className="text-yellow-300 text-sm">
                                        This is the only time you'll see this key. It has been automatically saved for this session.
                                    </p>
                                </div>
                            </div>
                        </div>

                        <div>
                            <label className="block text-gray-400 text-sm font-medium mb-2">
                                Your API Key
                            </label>
                            <div className="p-3 bg-slate-900 border border-slate-600 rounded-lg font-mono text-cyan-400 text-sm break-all">
                                {createdApiKey}
                            </div>
                        </div>

                        <button
                            onClick={() => {
                                setCreatedApiKey(null)
                                onClose()
                                // Reload to show authenticated state
                                window.location.reload()
                            }}
                            className="w-full py-2.5 bg-cyan-500 hover:bg-cyan-600 text-white font-semibold rounded-lg transition-colors"
                        >
                            Continue to Dashboard
                        </button>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
            <div className="bg-slate-800 border border-slate-700 rounded-xl w-full max-w-md shadow-2xl overflow-hidden">
                <div className="flex items-center justify-between p-6 border-b border-slate-700">
                    <h2 className="text-xl font-bold text-white">
                        {user ? 'Edit User' : 'Create New User'}
                    </h2>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-slate-700 rounded-lg text-gray-400 transition-colors"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <form onSubmit={handleSubmit} className="p-6 space-y-4">
                    <div>
                        <label className="block text-gray-400 text-sm font-medium mb-2">
                            Full Name
                        </label>
                        <input
                            type="text"
                            required
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="w-full px-4 py-2 bg-slate-900 border border-slate-600 rounded-lg text-white focus:border-cyan-500 focus:outline-none"
                            placeholder="John Doe"
                        />
                    </div>

                    <div>
                        <label className="block text-gray-400 text-sm font-medium mb-2">
                            Email Address
                        </label>
                        <input
                            type="email"
                            required
                            value={formData.email}
                            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                            className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:border-cyan-500 focus:outline-none disabled:opacity-50"
                            placeholder="john@example.com"
                        />
                    </div>

                    <div>
                        <label className="block text-gray-400 text-sm font-medium mb-2">
                            Role
                        </label>
                        <select
                            value={formData.role}
                            onChange={(e) =>
                                setFormData({ ...formData, role: e.target.value as UserRole })
                            }
                            className="w-full px-4 py-2 bg-slate-900 border border-slate-600 rounded-lg text-white focus:border-cyan-500 focus:outline-none"
                        >
                            <option value="member">Member</option>
                            <option value="admin">Admin</option>
                        </select>
                    </div>

                    {user && (
                        <div className="flex items-center gap-3 p-3 bg-slate-900/50 rounded-lg border border-slate-700">
                            <input
                                type="checkbox"
                                id="isActive"
                                checked={formData.isActive}
                                onChange={(e) =>
                                    setFormData({ ...formData, isActive: e.target.checked })
                                }
                                className="w-4 h-4 rounded border-slate-600 text-cyan-500 focus:ring-cyan-500 bg-slate-800"
                            />
                            <label htmlFor="isActive" className="text-sm font-medium text-gray-300">
                                Active Account
                            </label>
                        </div>
                    )}

                    <div className="flex gap-3 pt-4">
                        <button
                            type="submit"
                            disabled={isPending}
                            className="flex-1 py-2.5 bg-cyan-500 hover:bg-cyan-600 disabled:bg-slate-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors flex items-center justify-center gap-2"
                        >
                            {isPending && <Loader2 className="w-4 h-4 animate-spin" />}
                            {user ? 'Save Changes' : 'Create User'}
                        </button>
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-6 py-2.5 bg-slate-700 hover:bg-slate-600 text-gray-300 font-semibold rounded-lg transition-colors"
                        >
                            Cancel
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}

function UserManagementPage() {
    const { data: userData, isLoading } = useUsers()
    const [searchQuery, setSearchQuery] = useState('')
    const [modalOpen, setModalOpen] = useState(false)
    const [editingUser, setEditingUser] = useState<User | undefined>(undefined)

    const users = userData?.users || []

    // Simple client-side filtering
    const filteredUsers = users.filter(
        (u) =>
            u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            u.email.toLowerCase().includes(searchQuery.toLowerCase())
    )

    const handleEdit = (user: User) => {
        setEditingUser(user)
        setModalOpen(true)
    }

    const handleCreate = () => {
        setEditingUser(undefined)
        setModalOpen(true)
    }

    if (isLoading) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
                <div className="flex items-center gap-3 text-cyan-400">
                    <Loader2 className="w-6 h-6 animate-spin" />
                    <span>Loading users...</span>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
            <div className="max-w-6xl mx-auto">
                {/* Header */}
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8">
                    <div className="flex items-center gap-3">
                        <div className="p-3 bg-cyan-500/10 rounded-xl">
                            <UserIcon className="w-8 h-8 text-cyan-400" />
                        </div>
                        <div>
                            <h1 className="text-3xl font-bold text-white">Users</h1>
                            <p className="text-gray-400">Manage team members and permissions</p>
                        </div>
                    </div>
                    <button
                        onClick={handleCreate}
                        className="flex items-center gap-2 px-4 py-2 bg-cyan-500 hover:bg-cyan-600 text-white font-semibold rounded-lg transition-colors shadow-lg shadow-cyan-500/20"
                    >
                        <Plus className="w-5 h-5" />
                        Add User
                    </button>
                </div>

                {/* Filters */}
                <div className="mb-6">
                    <div className="relative max-w-md">
                        <Search className="absolute left-3 top-2.5 w-5 h-5 text-gray-500" />
                        <input
                            type="text"
                            placeholder="Search by name or email..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full pl-10 pr-4 py-2.5 bg-slate-800/50 border border-slate-700 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none transition-colors"
                        />
                    </div>
                </div>

                {/* Users Table */}
                <div className="bg-slate-800/50 border border-slate-700 rounded-xl overflow-hidden backdrop-blur-sm">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left">
                            <thead>
                                <tr className="bg-slate-900/50 border-b border-slate-700">
                                    <th className="px-6 py-4 text-gray-400 font-medium text-sm">User</th>
                                    <th className="px-6 py-4 text-gray-400 font-medium text-sm">Role</th>
                                    <th className="px-6 py-4 text-gray-400 font-medium text-sm">Status</th>
                                    <th className="px-6 py-4 text-gray-400 font-medium text-sm">Joined</th>
                                    <th className="px-6 py-4 text-gray-400 font-medium text-sm text-right">
                                        Actions
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-700">
                                {filteredUsers.length > 0 ? (
                                    filteredUsers.map((user) => (
                                        <tr
                                            key={user.id}
                                            className="group hover:bg-slate-800/50 transition-colors"
                                        >
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-3">
                                                    <div className="w-10 h-10 rounded-full bg-slate-700 flex items-center justify-center text-cyan-400 font-bold">
                                                        {user.name.charAt(0).toUpperCase()}
                                                    </div>
                                                    <div>
                                                        <p className="text-white font-medium">{user.name}</p>
                                                        <div className="flex items-center gap-1.5 text-sm text-gray-500">
                                                            <Mail className="w-3 h-3" />
                                                            {user.email}
                                                        </div>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-1.5">
                                                    <Shield className="w-4 h-4 text-slate-500" />
                                                    <span className="text-gray-300 capitalize">{user.role}</span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                {user.isActive ? (
                                                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20">
                                                        <CheckCircle className="w-3 h-3" />
                                                        Active
                                                    </span>
                                                ) : (
                                                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20">
                                                        <AlertCircle className="w-3 h-3" />
                                                        Inactive
                                                    </span>
                                                )}
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-1.5 text-gray-400 text-sm">
                                                    <Clock className="w-4 h-4 text-slate-500" />
                                                    {new Date(user.createdAt).toLocaleDateString()}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <button
                                                    onClick={() => handleEdit(user)}
                                                    className="p-2 hover:bg-slate-700 rounded-lg text-gray-400 hover:text-white transition-colors"
                                                    title="Edit User"
                                                >
                                                    <Edit2 className="w-4 h-4" />
                                                </button>
                                            </td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-12 text-center text-gray-500">
                                            No users found matching your search.
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                    <div className="px-6 py-4 border-t border-slate-700 bg-slate-900/30 text-sm text-gray-500">
                        Showing {filteredUsers.length} users
                    </div>
                </div>
            </div>

            <UserModal
                isOpen={modalOpen}
                onClose={() => setModalOpen(false)}
                user={editingUser}
            />
        </div>
    )
}
