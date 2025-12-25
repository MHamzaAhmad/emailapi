import { Link } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    PackageIcon,
    ZapIcon,
    Mail01Icon,
    Server,
    LockKeyIcon
} from '@hugeicons/core-free-icons'
import { motion, AnimatePresence } from 'framer-motion'
import { useState } from 'react'

export function Features() {
    const [isHoveringEmail, setIsHoveringEmail] = useState(false)

    return (
        <section className="py-24 px-6 md:px-8">
            <div className="mx-auto max-w-6xl">
                <div className="mb-16 md:text-center max-w-3xl mx-auto">
                    <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-6">
                        Developers First, <span className="text-primary">Always</span>
                    </h2>
                    <p className="text-muted-foreground text-lg leading-relaxed">
                        We handle the heavy lifting of email infrastructure so you can ship features faster.
                    </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {/* Feature 1: Type Safety */}
                    <div className="group md:col-span-2 relative overflow-visible rounded-xl border border-white/5 bg-zinc-950 p-8 hover:border-white/10 transition-colors z-20">
                        <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity rounded-xl pointer-events-none" />
                        <div className="relative z-10 flex flex-col sm:flex-row h-full gap-8">
                            <div className="flex-1 space-y-4">
                                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-zinc-900 border border-white/5 text-primary">
                                    <HugeiconsIcon icon={PackageIcon} size={20} />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold tracking-tight mb-2 text-white">Full Type Safety</h3>
                                    <p className="text-zinc-400 leading-relaxed text-sm">
                                        Never guess what an event looks like. Our SDKs provide full autocomplete and type checks for every webhook and event callback.
                                    </p>
                                </div>
                            </div>
                            {/* Visual: IDE Hint */}
                            <div className="flex-1 rounded-lg bg-black/50 border border-white/5 p-4 font-mono text-xs shadow-inner relative overflow-visible">
                                <div className="text-zinc-500 mb-2">// Fully typed event objects</div>
                                <div className="text-violet-400">client<span className="text-zinc-300">.</span>onReceive<span className="text-zinc-300">({'{'}</span></div>
                                <div className="pl-4 text-sky-300">onReplied<span className="text-zinc-300">: (</span>event<span className="text-zinc-300">) </span><span className="text-violet-400">=&gt;</span> <span className="text-zinc-300">{'{'}</span></div>
                                <div className="pl-8 text-zinc-300">
                                    <span className="text-zinc-500">// Hover below to see types</span>
                                </div>
                                <div className="pl-8 text-zinc-300 relative">
                                    console<span className="text-zinc-300">.</span>log<span className="text-zinc-300">(</span>event<span className="text-zinc-300">.</span>
                                    <span
                                        className="text-white underline decoration-dashed decoration-zinc-600 underline-offset-4 cursor-help relative inline-block"
                                        onMouseEnter={() => setIsHoveringEmail(true)}
                                        onMouseLeave={() => setIsHoveringEmail(false)}
                                    >
                                        email

                                        {/* Autocomplete Popover */}
                                        <AnimatePresence>
                                            {isHoveringEmail && (
                                                <motion.div
                                                    initial={{ opacity: 0, y: 5, scale: 0.95 }}
                                                    animate={{ opacity: 1, y: 0, scale: 1 }}
                                                    exit={{ opacity: 0, y: 5, scale: 0.95 }}
                                                    transition={{ duration: 0.1 }}
                                                    className="absolute left-0 top-full mt-2 w-48 bg-[#1e1e1e] border border-[#333] shadow-2xl rounded-md overflow-hidden z-50 text-xs font-mono"
                                                >
                                                    <div className="flex items-center px-2 py-1.5 border-b border-[#333] bg-[#252526] text-zinc-400 text-[10px] uppercase tracking-wider">
                                                        Property
                                                    </div>
                                                    <div className="py-1">
                                                        <div className="px-3 py-1.5 hover:bg-[#2a2d2e] cursor-pointer flex items-center justify-between group/item">
                                                            <span className="text-sky-300">subject</span>
                                                            <span className="text-zinc-500 text-[10px]">string</span>
                                                        </div>
                                                        <div className="px-3 py-1.5 hover:bg-[#2a2d2e] cursor-pointer flex items-center justify-between group/item bg-[#094771] text-white">
                                                            <span className="text-white">body</span>
                                                            <span className="text-white/70 text-[10px]">string</span>
                                                        </div>
                                                        <div className="px-3 py-1.5 hover:bg-[#2a2d2e] cursor-pointer flex items-center justify-between group/item">
                                                            <span className="text-sky-300">from</span>
                                                            <span className="text-zinc-500 text-[10px]">string</span>
                                                        </div>
                                                        <div className="px-3 py-1.5 hover:bg-[#2a2d2e] cursor-pointer flex items-center justify-between group/item">
                                                            <span className="text-sky-300">to</span>
                                                            <span className="text-zinc-500 text-[10px]">string[]</span>
                                                        </div>
                                                        <div className="px-3 py-1.5 hover:bg-[#2a2d2e] cursor-pointer flex items-center justify-between group/item">
                                                            <span className="text-sky-300">attachments</span>
                                                            <span className="text-zinc-500 text-[10px]">Attachment[]</span>
                                                        </div>
                                                    </div>
                                                </motion.div>
                                            )}
                                        </AnimatePresence>
                                    </span>
                                    <span className="text-zinc-300">)</span>
                                </div>
                                <div className="pl-4 text-zinc-300">{'}'}</div>
                                <div className="text-zinc-300">{'}'})</div>
                            </div>
                        </div>
                    </div>

                    {/* Feature 2: Real-time (Animated) */}
                    <div className="group md:col-span-1 md:row-span-2 relative overflow-hidden rounded-xl border border-white/5 bg-zinc-950 p-8 hover:border-white/10 transition-colors flex flex-col">
                        <div className="absolute inset-0 bg-gradient-to-b from-primary/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                        <div className="relative z-10 flex flex-col flex-1 gap-8">
                            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-zinc-900 border border-white/5 text-orange-400">
                                <HugeiconsIcon icon={ZapIcon} size={20} />
                            </div>
                            <div>
                                <h3 className="text-xl font-bold tracking-tight mb-2 text-white">Real-time Events</h3>
                                <p className="text-zinc-400 leading-relaxed text-sm mb-3">
                                    Receive delivery updates and inbound replies instantly via reliable webhooks or long-lived callbacks.
                                </p>
                                {/* Micro-Animation: Event Stream */}
                                {/* Increased height to h-[180px] to prevent cropping */}
                                <div className="mt-8 space-y-3 font-mono text-xs overflow-hidden h-[180px] relative mask-linear-gradient">
                                    <EventStream />
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Feature 3: Smart Queues */}
                    <div className="group md:col-span-1 relative overflow-hidden rounded-xl border border-white/5 bg-zinc-950 p-8 hover:border-white/10 transition-colors">
                        <div className="absolute inset-0 bg-gradient-to-tr from-primary/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                        <div className="relative z-10 space-y-4">
                            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-zinc-900 border border-white/5 text-blue-400">
                                <HugeiconsIcon icon={Mail01Icon} size={20} />
                            </div>
                            <div>
                                <h3 className="text-lg font-bold tracking-tight mb-2 text-white">Smart Queuing</h3>
                                <p className="text-zinc-400 text-sm leading-relaxed">
                                    Fire and forget. We intelligently queue your emails, handling retries, rate limits, and backpressure so you don't have to.
                                </p>
                            </div>
                        </div>
                    </div>

                    {/* Feature 4: Performance */}
                    <div className="group md:col-span-1 relative overflow-hidden rounded-xl border border-white/5 bg-zinc-950 p-8 hover:border-white/10 transition-colors">
                        <div className="absolute inset-0 bg-gradient-to-tl from-primary/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                        <div className="relative z-10 space-y-4">
                            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-zinc-900 border border-white/5 text-emerald-400">
                                <HugeiconsIcon icon={Server} size={20} />
                            </div>
                            <div>
                                <h3 className="text-lg font-bold tracking-tight mb-2 text-white">Global Low Latency</h3>
                                <p className="text-zinc-400 text-sm leading-relaxed">
                                    Built on HTTP/2 for ultra-low latency connections. We manage the complexity of the underlying email infrastructure while you get raw speed.
                                </p>
                            </div>
                        </div>
                    </div>

                </div>
            </div>
        </section>
    )
}

function EventStream() {
    const events = [
        { type: 'email.delivered', time: '10:42:01', color: 'text-emerald-400', bg: 'bg-emerald-500' },
        { type: 'email.replied', time: '10:42:05', color: 'text-blue-400', bg: 'bg-blue-500' },
        { type: 'email.clicked', time: '10:42:08', color: 'text-purple-400', bg: 'bg-purple-500' },
        { type: 'email.bounced', time: '10:42:15', color: 'text-rose-400', bg: 'bg-rose-500' },
    ]

    return (
        <div className="flex flex-col gap-3">
            {events.map((event, i) => (
                <motion.div
                    key={event.type}
                    initial={{ opacity: 0, x: -10 }}
                    whileInView={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.5, duration: 0.5 }}
                    className={`flex items-center gap-3 ${event.color}`}
                >
                    <div className={`h-1.5 w-1.5 rounded-full ${event.bg} shadow-[0_0_8px_currentColor]`} />
                    <span className="opacity-90">{event.type}</span>
                    <span className="text-zinc-600 ml-auto">{event.time}</span>
                </motion.div>
            ))}
            <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: [0, 1, 0] }}
                transition={{ duration: 2, repeat: Infinity, delay: 2 }}
                className="flex items-center gap-3 text-zinc-500"
            >
                <div className="h-1.5 w-1.5 rounded-full bg-zinc-700" />
                <span>waiting for event...</span>
            </motion.div>
        </div>
    )
}
