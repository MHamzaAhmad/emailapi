import { createFileRoute } from '@tanstack/react-router'
import { PackageIcon, Download02Icon, Settings04Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useState, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

export const Route = createFileRoute('/assets')({
    component: AssetsPage,
})

type AssetType = 'logo' | 'icon' | 'header'
type PlatformPreset = 'twitter' | 'linkedin_profile' | 'linkedin_company' | 'open_graph' | 'custom'
type LogoSymbol = 'flow' | 'exchange' | 'plane' | 'bolt' | 'blueprint'
type HeaderStyle = 'minimal' | 'gradient' | 'developer'

const PRESETS = {
    twitter: { w: 1500, h: 500, label: 'Twitter Header (1500x500)' },
    linkedin_profile: { w: 1584, h: 396, label: 'LinkedIn Profile (1584x396)' },
    linkedin_company: { w: 1128, h: 191, label: 'LinkedIn Company (1128x191)' },
    open_graph: { w: 1200, h: 630, label: 'Open Graph (1200x630)' },
    custom: { w: 1200, h: 630, label: 'Custom' },
}

function AssetsPage() {
    // State
    const [assetType, setAssetType] = useState<AssetType>('header')
    const [preset, setPreset] = useState<PlatformPreset>('twitter')
    const [symbol, setSymbol] = useState<LogoSymbol>('blueprint')
    const [headerStyle, setHeaderStyle] = useState<HeaderStyle>('developer')
    const [width, setWidth] = useState(PRESETS.twitter.w)
    const [height, setHeight] = useState(PRESETS.twitter.h)

    // Tagline State
    const [tagline, setTagline] = useState("The simplest API to send and receive emails.")
    const [showTagline, setShowTagline] = useState(false)

    // Style State
    const [theme, setTheme] = useState<'light' | 'dark'>('light') // Dark means Dark Background
    const [transparent, setTransparent] = useState(false)
    const [showText, setShowText] = useState(true)
    const [showIcon, setShowIcon] = useState(true)

    const svgRef = useRef<SVGSVGElement>(null)

    // Update dimensions when preset changes
    useEffect(() => {
        if (preset !== 'custom') {
            setWidth(PRESETS[preset].w)
            setHeight(PRESETS[preset].h)
        }
    }, [preset])

    // Update dimensions when type changes
    useEffect(() => {
        if (assetType === 'icon') {
            setWidth(512)
            setHeight(512)
            setShowText(false)
            setShowIcon(true)
            setShowTagline(false)
        } else if (assetType === 'logo') {
            setWidth(1200) // Generic nice width
            setHeight(300)
            setShowText(true)
            setShowIcon(true)
            setShowTagline(false)
        } else if (assetType === 'header') {
            // Revert to current preset
            if (preset !== 'custom') {
                setWidth(PRESETS[preset].w)
                setHeight(PRESETS[preset].h)
            }
            setShowText(true)
            setShowIcon(true)
            setShowTagline(true)
        }
    }, [assetType])


    const download = (format: 'svg' | 'png') => {
        if (!svgRef.current) return

        if (format === 'svg') {
            const svgData = new XMLSerializer().serializeToString(svgRef.current)
            const blob = new Blob([svgData], { type: 'image/svg+xml;charset=utf-8' })
            const url = URL.createObjectURL(blob)
            const link = document.createElement('a')
            link.href = url
            link.download = `simple-email-api-${assetType}-${theme}.${format}`
            document.body.appendChild(link)
            link.click()
            document.body.removeChild(link)
        } else {
            const svgData = new XMLSerializer().serializeToString(svgRef.current)
            const img = new Image()
            const blob = new Blob([svgData], { type: 'image/svg+xml;charset=utf-8' })
            const url = URL.createObjectURL(blob)

            img.onload = () => {
                const canvas = document.createElement('canvas')
                canvas.width = width
                canvas.height = height
                const ctx = canvas.getContext('2d')
                if (!ctx) return

                ctx.drawImage(img, 0, 0)
                const pngUrl = canvas.toDataURL('image/png')

                const link = document.createElement('a')
                link.href = pngUrl
                link.download = `simple-email-api-${assetType}-${theme}.png`
                document.body.appendChild(link)
                link.click()
                document.body.removeChild(link)
                URL.revokeObjectURL(url)
            }
            img.src = url
        }
    }

    // Colors
    const bgColor = transparent ? 'transparent' : (theme === 'dark' ? '#09090b' : '#ffffff')
    const fgColor = theme === 'dark' ? '#ffffff' : '#09090b'

    // Syntax Highlight Colors
    const sh = {
        keyword: theme === 'dark' ? '#C678DD' : '#A626A4', // Purple
        string: theme === 'dark' ? '#98C379' : '#50A14F',  // Green
        func: theme === 'dark' ? '#61AFEF' : '#4078F2',    // Blue
        plain: fgColor
    }
    // const accentColor = '#10b981' // emerald-500

    // Grid Background for Header visual interest (optional)
    const showGrid = assetType === 'header' && !transparent

    return (
        <div className="min-h-screen bg-background text-foreground font-sans flex flex-col">
            {/* Navigation Stub */}
            <nav className="border-b border-dashed border-border/40 bg-background/80 backdrop-blur-xl sticky top-0 z-50">
                <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-6">
                    <a href="/dashboard" className="flex items-center gap-2 font-bold tracking-tight text-foreground/90 hover:opacity-80 transition-opacity">
                        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-primary-foreground shadow-sm">
                            <HugeiconsIcon icon={PackageIcon} size={14} strokeWidth={2.5} />
                        </div>
                        <span className="text-sm">SimpleEmailAPI</span>
                    </a>
                    <span className="text-xs font-mono text-muted-foreground uppercase tracking-wider">Asset Generator</span>
                </div>
            </nav>

            <main className="flex-1 flex flex-col md:flex-row h-[calc(100vh-3.5rem)] overflow-hidden">

                {/* Controls Sidebar */}
                <aside className="w-full md:w-80 border-b md:border-b-0 md:border-r border-dashed border-border/40 bg-card/50 p-6 flex flex-col gap-8 overflow-y-auto">

                    <div className="space-y-4">
                        <div className="flex items-center gap-2 text-sm font-bold uppercase tracking-widest text-muted-foreground mb-2">
                            <HugeiconsIcon icon={Settings04Icon} size={16} />
                            Configuration
                        </div>

                        <div className="space-y-3">
                            <Label>Asset Type</Label>
                            <Tabs value={assetType} onValueChange={(v) => setAssetType(v as any)} className="w-full">
                                <TabsList className="w-full grid grid-cols-3">
                                    <TabsTrigger value="logo">Logo</TabsTrigger>
                                    <TabsTrigger value="icon">Icon</TabsTrigger>
                                    <TabsTrigger value="header">Header</TabsTrigger>
                                </TabsList>
                            </Tabs>
                        </div>

                        <div className="space-y-3">
                            <Label>Symbol Concept</Label>
                            <Select value={symbol} onValueChange={(v) => setSymbol(v as any)}>
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="blueprint">Protocol (Bespoke)</SelectItem>
                                    <SelectItem value="flow">Flow (In/Out & Speed)</SelectItem>
                                    <SelectItem value="exchange">Exchange (Minimal Arrows)</SelectItem>
                                    <SelectItem value="plane">Paper Plane (Send)</SelectItem>
                                    <SelectItem value="bolt">Lightning (Speed)</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>

                        {assetType === 'header' && (
                            <div className="space-y-3 animate-in fade-in slide-in-from-top-2">
                                <Label>Header Style</Label>
                                <Select value={headerStyle} onValueChange={(v) => setHeaderStyle(v as any)}>
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="developer">Developer (Code Snippet)</SelectItem>
                                        <SelectItem value="gradient">Modern Gradient</SelectItem>
                                        <SelectItem value="minimal">Minimal Grid</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                        )}

                        {assetType === 'header' && (
                            <div className="space-y-3 animate-in fade-in slide-in-from-top-2">
                                <Label>Platform Preset</Label>
                                <Select value={preset} onValueChange={(v) => setPreset(v as any)}>
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {Object.entries(PRESETS).map(([key, val]) => (
                                            <SelectItem key={key} value={key}>{val.label}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                        )}

                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <Label>Width</Label>
                                <Input type="number" value={width} onChange={e => setWidth(Number(e.target.value))} disabled={preset !== 'custom' && assetType === 'header'} />
                            </div>
                            <div className="space-y-2">
                                <Label>Height</Label>
                                <Input type="number" value={height} onChange={e => setHeight(Number(e.target.value))} disabled={preset !== 'custom' && assetType === 'header'} />
                            </div>
                        </div>
                    </div>

                    <div className="space-y-4">
                        <Label>Appearance</Label>

                        <div className="flex items-center justify-between p-3 rounded-lg border border-border/40 bg-background">
                            <Label htmlFor="theme-toggle" className="cursor-pointer">Dark Mode</Label>
                            <Switch id="theme-toggle" checked={theme === 'dark'} onCheckedChange={(c) => setTheme(c ? 'dark' : 'light')} />
                        </div>

                        <div className="flex items-center justify-between p-3 rounded-lg border border-border/40 bg-background">
                            <Label htmlFor="transparent-toggle" className="cursor-pointer">Transparent Background</Label>
                            <Switch id="transparent-toggle" checked={transparent} onCheckedChange={setTransparent} />
                        </div>

                        <div className="space-y-3 pt-2">
                            <div className="flex items-center gap-2">
                                <Switch id="show-icon" checked={showIcon} onCheckedChange={setShowIcon} />
                                <Label htmlFor="show-icon">Show Icon</Label>
                            </div>
                            <div className="flex items-center gap-2">
                                <Switch id="show-text" checked={showText} onCheckedChange={setShowText} />
                                <Label htmlFor="show-text">Show Text</Label>
                            </div>
                        </div>

                        {assetType === 'header' && (
                            <div className="space-y-3 pt-4 border-t border-dashed border-border/40">
                                <div className="flex items-center justify-between">
                                    <Label htmlFor="show-tagline" className="cursor-pointer">Tagline</Label>
                                    <Switch id="show-tagline" checked={showTagline} onCheckedChange={setShowTagline} />
                                </div>
                                {showTagline && (
                                    <Input
                                        value={tagline}
                                        onChange={(e) => setTagline(e.target.value)}
                                        placeholder="Enter tagline..."
                                        className="text-xs"
                                    />
                                )}
                            </div>
                        )}
                    </div>

                    <div className="mt-auto space-y-3 pt-6 border-t border-dashed border-border/40">
                        <Button onClick={() => download('png')} className="w-full gap-2" size="lg">
                            <HugeiconsIcon icon={Download02Icon} size={18} />
                            Download PNG
                        </Button>
                        <Button onClick={() => download('svg')} variant="outline" className="w-full gap-2 border-dashed" size="lg">
                            <HugeiconsIcon icon={Download02Icon} size={18} />
                            Download SVG
                        </Button>
                    </div>

                </aside>

                {/* Preview Area */}
                <div className="flex-1 bg-secondary/20 flex items-center justify-center p-8 md:p-12 overflow-hidden relative">

                    {/* Checkerboard for transparency */}
                    <div className="absolute inset-0 z-0 opacity-20 pointer-events-none"
                        style={{
                            backgroundImage: `linear-gradient(45deg, #808080 25%, transparent 25%), linear-gradient(-45deg, #808080 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #808080 75%), linear-gradient(-45deg, transparent 75%, #808080 75%)`,
                            backgroundSize: '20px 20px',
                            backgroundPosition: '0 0, 0 10px, 10px -10px, -10px 0px'
                        }}
                    />

                    <div className="relative z-10 shadow-2xl transition-all duration-300" style={{ maxWidth: '100%', maxHeight: '100%' }}>
                        {/* SVG RENDERER */}
                        <svg
                            ref={svgRef}
                            width={width}
                            height={height}
                            viewBox={`0 0 ${width} ${height}`}
                            xmlns="http://www.w3.org/2000/svg"
                            style={{
                                width: assetType === 'icon' ? 300 : (width > 800 ? '100%' : width),
                                height: 'auto',
                                maxWidth: '100%'
                            }}
                        >
                            {/* Background */}
                            <rect width="100%" height="100%" fill={bgColor} />

                            {/* Styles Backgrounds */}
                            {assetType === 'header' && headerStyle === 'gradient' && (
                                <g>
                                    <defs>
                                        <radialGradient id="glow" cx="50%" cy="50%" r="50%" fx="50%" fy="50%">
                                            <stop offset="0%" stopColor={fgColor} stopOpacity="0.15" />
                                            <stop offset="100%" stopColor={fgColor} stopOpacity="0" />
                                        </radialGradient>
                                    </defs>
                                    <rect width="100%" height="100%" fill="url(#glow)" />
                                </g>
                            )}

                            {assetType === 'header' && headerStyle === 'minimal' && showGrid && (
                                <g opacity="0.1">
                                    <defs>
                                        <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                                            <path d="M 40 0 L 0 0 0 40" fill="none" stroke={fgColor} strokeWidth="1" />
                                        </pattern>
                                    </defs>
                                    <rect width="100%" height="100%" fill="url(#grid)" />
                                </g>
                            )}

                            {assetType === 'header' && headerStyle === 'developer' && (
                                <g
                                    opacity={1}
                                    transform={`translate(${width * 0.52}, ${height * 0.5 - (250 * Math.min(width / 1500, height / 750))}) scale(${Math.min(width / 1500, height / 750)})`}
                                >
                                    <rect x="-20" y="-20" width="700" height="500" rx="12" fill={fgColor} opacity="0.05" />
                                    <g transform="translate(20, 30)">
                                        <circle cx="0" cy="0" r="6" fill="#FF5F56" />
                                        <circle cx="20" cy="0" r="6" fill="#FFBD2E" />
                                        <circle cx="40" cy="0" r="6" fill="#27C93F" />
                                    </g>

                                    <text x="0" y="80" fontFamily="Monaco, monospace" fontSize="20" fill={sh.plain} fontWeight="bold">
                                        <tspan x="0" dy="0">&gt; npm i simpleemailapi</tspan>
                                    </text>

                                    <text x="0" y="140" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        <tspan x="0" dy="0" fill={sh.keyword}>import</tspan> {`{`} <tspan fill={sh.func}>createClient</tspan> {`}`} <tspan fill={sh.keyword}>from</tspan> <tspan fill={sh.string}>'simpleemailapi'</tspan>
                                    </text>
                                    <text x="0" y="180" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        <tspan x="0" dy="0" fill={sh.keyword}>const</tspan> client = <tspan fill={sh.func}>createClient</tspan>(...)
                                    </text>

                                    <text x="0" y="240" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        <tspan x="0" dy="0" fill={sh.keyword}>await</tspan> client.<tspan fill={sh.func}>send</tspan>({`{`} to: <tspan fill={sh.string}>'user@ex.com'</tspan>, body: ... {`}`})
                                    </text>

                                    <text x="0" y="300" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        client.<tspan fill={sh.func}>onReceive</tspan>({`{`}
                                    </text>
                                    <text x="30" y="330" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        <tspan fill={sh.func}>onReplied</tspan>: (email) ={`>`} {`{`}
                                    </text>
                                    <text x="60" y="360" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        console.<tspan fill={sh.func}>log</tspan>(<tspan fill={sh.string}>'New reply:'</tspan>, email)
                                    </text>
                                    <text x="30" y="390" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        {`}`},
                                    </text>
                                    <text x="0" y="420" fontFamily="Monaco, monospace" fontSize="18" fill={sh.plain} style={{ whiteSpace: 'pre' }}>
                                        {`}`})
                                    </text>
                                </g>
                            )}

                            {/* Content Group - Centered */}
                            <g transform={`translate(${assetType === 'header' && headerStyle === 'developer' ? width * 0.25 : width / 2}, ${height / 2}) scale(${assetType === 'header' && headerStyle === 'developer' ? 0.95 : 1})`}>
                                <g transform={`translate(0, ${showTagline && assetType === 'header' ? -25 : 0})`}>
                                    <LogoContent
                                        symbol={symbol}
                                        fgColor={fgColor}
                                        showIcon={showIcon}
                                        showText={showText}
                                        layout={assetType === 'icon' ? 'stacked' : 'horizontal'}
                                        theme={theme}
                                    />
                                </g>

                                {showTagline && assetType === 'header' && (
                                    <text
                                        y={60}
                                        fill={fgColor}
                                        opacity={0.7}
                                        fontFamily="Inter, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
                                        fontSize={24}
                                        fontWeight="500"
                                        letterSpacing="-0.01em"
                                        textAnchor="middle"
                                    >
                                        {tagline}
                                    </text>
                                )}
                            </g>
                        </svg>
                    </div>
                </div>
            </main>
        </div>
    )
}

function LogoContent({ symbol, fgColor, showIcon, showText, layout, theme }: { symbol: LogoSymbol, fgColor: string, showIcon: boolean, showText: boolean, layout: 'horizontal' | 'stacked', theme: 'light' | 'dark' }) {

    // Scale factor to make things fit nicely
    const scale = layout === 'stacked' ? 4 : 2.5

    // Layout Logic
    // If only icon: center icon
    // If only text: center text
    // If both: 
    //    Horizontal: Icon | Gap | Text
    //    Stacked: Icon / Gap / Text

    const iconSize = 24 * scale
    const fontSize = 16 * scale
    const gap = 12 * scale

    // Calculate offets to center the group
    let totalWidth = 0
    let totalHeight = 0

    // Simple text width approx (Inter font is roughly 0.6em wide per char on averge, bold)
    // "SimpleEmailAPI" = 14 chars. 
    // 14 * 0.6 * fontSize = width
    const textWidth = showText ? (14 * 0.63 * fontSize) : 0
    const textHeight = fontSize // Cap height approx

    if (layout === 'horizontal') {
        if (showIcon && showText) {
            totalWidth = iconSize + gap + textWidth
            totalHeight = Math.max(iconSize, textHeight)
        } else if (showIcon) {
            totalWidth = iconSize
            totalHeight = iconSize
        } else if (showText) {
            totalWidth = textWidth
            totalHeight = textHeight
        }
    } else {
        // Stacked (mostly for icon only really, or vertical logo)
        if (showIcon && showText) {
            totalWidth = Math.max(iconSize, textWidth)
            totalHeight = iconSize + gap + textHeight
        } else if (showIcon) {
            totalWidth = iconSize
            totalHeight = iconSize
        } else if (showText) {
            totalWidth = textWidth
            totalHeight = textHeight
        }
    }

    const startX = -totalWidth / 2
    const startY = -totalHeight / 2 // Top of the bounding box relative to center

    return (
        <g>
            {showIcon && (
                <g transform={`translate(${layout === 'horizontal'
                    ? startX + (showText ? 0 : (totalWidth - iconSize) / 2)
                    : -iconSize / 2 // Centered horizontally
                    }, ${layout === 'horizontal'
                        ? -iconSize / 2 // Centered vertically
                        : startY      // Top
                    })`}>

                    {(!symbol || symbol === 'blueprint') && (
                        <g transform={`scale(${scale})`} shapeRendering="geometricPrecision">
                            {/* "The Protocol" Refined - Clean Pointed Tip */}

                            {/* Unified Body Path - Smooth unified shape */}
                            <path d="M22 2L2 10L10 13L14 22L22 2Z" stroke={fgColor} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" fill="none" />

                            {/* Central Crease - Defines the fold */}
                            <path d="M10 13L22 2" stroke={fgColor} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" fill="none" opacity="0.5" />
                        </g>
                    )}

                    {symbol === 'flow' && (
                        <g transform={`scale(${scale})`}>
                            {/* Inbound Arrow (Top Right to Bottom Leftish) */}
                            <path d="M20 7L13 14" stroke={fgColor} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
                            <path d="M13 14H17M13 14V10" stroke={fgColor} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />

                            {/* Outbound Arrow (Bottom Left to Top Rightish) */}
                            <path d="M4 17L11 10" stroke={fgColor} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
                            <path d="M11 10V14M11 10H7" stroke={fgColor} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
                        </g>
                    )}

                    {symbol === 'exchange' && (
                        <g transform={`scale(${scale})`}>
                            {/* Minimal Up/Down arrows */}
                            <path d="M7 17V7M7 7L3 11M7 7L11 11" stroke={fgColor} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                            <path d="M17 7V17M17 17L21 13M17 17L13 13" stroke={fgColor} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                        </g>
                    )}

                    {symbol === 'plane' && (
                        <g transform={`scale(${scale})`}>
                            <path d="M2 12L22 2L12 22L10 14L2 12Z" stroke={fgColor} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" fill="none" />
                            <path d="M22 2L10 14" stroke={fgColor} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                        </g>
                    )}

                    {symbol === 'bolt' && (
                        <g transform={`scale(${scale})`}>
                            <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke={fgColor} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" fill="none" />
                        </g>
                    )}

                </g>
            )}

            {showText && (
                <text
                    x={layout === 'horizontal' ? startX + (showIcon ? iconSize + gap : 0) : 0}
                    y={layout === 'horizontal' ? fontSize * 0.35 : startY + iconSize + gap + fontSize}
                    fill={fgColor}
                    fontFamily="Inter, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif"
                    fontWeight="800"
                    fontSize={fontSize}
                    letterSpacing="-0.03em"
                    textAnchor={layout === 'horizontal' ? "start" : "middle"}
                >
                    SimpleEmailAPI
                </text>
            )}
        </g>
    )
}
