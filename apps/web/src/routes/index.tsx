import { createFileRoute, Link } from '@tanstack/react-router'
import { Key, Globe, User, ArrowRight } from 'lucide-react'

export const Route = createFileRoute('/')({
  component: Dashboard,
})

function Dashboard() {
  const quickActions = [
    {
      icon: <User className="w-8 h-8 text-cyan-400" />,
      title: 'User Management',
      description: 'Create and manage your user account',
      href: '/user',
    },
    {
      icon: <Key className="w-8 h-8 text-cyan-400" />,
      title: 'API Keys',
      description: 'Create and manage API keys for authentication',
      href: '/api-keys',
    },
    {
      icon: <Globe className="w-8 h-8 text-cyan-400" />,
      title: 'Domains',
      description: 'Configure sending domains and DNS records',
      href: '/domains',
    },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900">
      <section className="relative py-20 px-6 text-center overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-r from-cyan-500/10 via-blue-500/10 to-purple-500/10" />
        <div className="relative max-w-5xl mx-auto">
          <h1 className="text-5xl md:text-6xl font-black text-white mb-4">
            <span className="bg-gradient-to-r from-cyan-400 to-blue-400 bg-clip-text text-transparent">
              Email API
            </span>{' '}
            <span className="text-gray-300">Dashboard</span>
          </h1>
          <p className="text-xl md:text-2xl text-gray-400 max-w-3xl mx-auto mb-8">
            Manage your sending domains, API keys, and email delivery
          </p>
        </div>
      </section>

      <section className="py-12 px-6 max-w-5xl mx-auto">
        <h2 className="text-2xl font-bold text-white mb-6">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {quickActions.map((action) => (
            <Link
              key={action.href}
              to={action.href}
              className="group bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-xl p-6 hover:border-cyan-500/50 transition-all duration-300 hover:shadow-lg hover:shadow-cyan-500/10"
            >
              <div className="mb-4">{action.icon}</div>
              <h3 className="text-xl font-semibold text-white mb-2 flex items-center gap-2">
                {action.title}
                <ArrowRight className="w-4 h-4 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all" />
              </h3>
              <p className="text-gray-400">{action.description}</p>
            </Link>
          ))}
        </div>
      </section>

      <section className="py-12 px-6 max-w-5xl mx-auto">
        <div className="bg-slate-800/30 border border-slate-700 rounded-xl p-8">
          <h2 className="text-xl font-bold text-white mb-4">Getting Started</h2>
          <div className="space-y-4 text-gray-400">
            <p>
              <span className="text-cyan-400 font-mono">1.</span> Create a user
              account to get started
            </p>
            <p>
              <span className="text-cyan-400 font-mono">2.</span> Generate an API
              key to authenticate your requests
            </p>
            <p>
              <span className="text-cyan-400 font-mono">3.</span> Add and verify
              your sending domain
            </p>
            <p>
              <span className="text-cyan-400 font-mono">4.</span> Start sending
              emails using the API
            </p>
          </div>
        </div>
      </section>
    </div>
  )
}
