import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { Key, Globe, User, ArrowRight, Zap, CheckCircle2 } from 'lucide-react'
import { useEffect } from 'react'
import { getAuthToken } from '@/lib/api'
import { useCurrentUser } from '@/hooks'

export const Route = createFileRoute('/')({
  component: Dashboard,
})

function Dashboard() {
  const navigate = useNavigate()
  const token = getAuthToken()
  const { data: user } = useCurrentUser()

  useEffect(() => {
    if (!token) {
      navigate({ to: '/user' })
    }
  }, [token, navigate])

  if (!token) return null

  const quickActions = [
    {
      icon: <Globe className="w-8 h-8 text-cyan-400" />,
      title: 'Domains',
      description: 'Configure and verify sending domains',
      href: '/domains',
      color: 'cyan',
    },
    {
      icon: <Key className="w-8 h-8 text-purple-400" />,
      title: 'API Keys',
      description: 'Manage authentication tokens',
      href: '/api-keys',
      color: 'purple',
    },
    {
      icon: <User className="w-8 h-8 text-indigo-400" />,
      title: 'Session',
      description: `Logged in as ${user?.name || 'User'}`,
      href: '/user',
      color: 'indigo',
    },
  ]

  return (
    <div className="min-h-screen bg-slate-950 text-white p-6 md:p-12">
      <div className="max-w-5xl mx-auto space-y-12">
        {/* Header */}
        <div className="space-y-4">
          <div className="flex items-center gap-3 text-cyan-400 mb-2">
            <Zap className="w-6 h-6" />
            <span className="font-mono text-sm tracking-wide uppercase">Email API Test Console</span>
          </div>
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight">
            Ready to Build.
          </h1>
          <p className="text-xl text-slate-400 max-w-2xl">
            You are authenticated and connected to the local API environment.
          </p>

          <div className="inline-flex items-center gap-2 px-3 py-1 bg-green-500/10 text-green-400 rounded-full text-sm font-medium border border-green-500/20">
            <CheckCircle2 className="w-4 h-4" />
            System Operational
          </div>
        </div>

        {/* Quick Actions Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {quickActions.map((action) => (
            <Link
              key={action.href}
              to={action.href}
              className="group relative bg-slate-900 border border-slate-800 rounded-2xl p-6 hover:border-slate-700 transition-all hover:-translate-y-1 hover:shadow-xl"
            >
              <div className={`mb-4 p-3 rounded-xl bg-${action.color}-500/10 w-fit`}>
                {action.icon}
              </div>
              <h3 className="text-lg font-bold mb-2 flex items-center gap-2">
                {action.title}
                <ArrowRight className="w-4 h-4 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all text-slate-400" />
              </h3>
              <p className="text-slate-400 leading-relaxed">
                {action.description}
              </p>
            </Link>
          ))}
        </div>

        {/* Status Section */}
        <div className="grid md:grid-cols-2 gap-6">
          <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6">
            <h3 className="font-semibold text-slate-200 mb-4">Environment</h3>
            <div className="space-y-3 font-mono text-sm">
              <div className="flex justify-between">
                <span className="text-slate-500">API Endpoint</span>
                <span className="text-cyan-400">http://localhost:8080</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Gateway</span>
                <span className="text-purple-400">gRPC-Gateway</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Version</span>
                <span className="text-slate-200">v1.0.0-dev</span>
              </div>
            </div>
          </div>
          <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6">
            <h3 className="font-semibold text-slate-200 mb-4">Current Session</h3>
            <div className="space-y-3 font-mono text-sm">
              <div className="flex justify-between">
                <span className="text-slate-500">User ID</span>
                <span className="text-slate-200" title={user?.id}>{user?.id?.slice(0, 12)}...</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Email</span>
                <span className="text-slate-200">{user?.email}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Role</span>
                <span className="text-indigo-400">{user?.role}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
