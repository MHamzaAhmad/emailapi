import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { HugeiconsIcon } from '@hugeicons/react'
import { HelpCircleIcon } from '@hugeicons/core-free-icons'
import { cn } from '@/lib/utils'

interface InfoTooltipProps {
    children?: React.ReactNode
    content: string
    className?: string
}

export function InfoTooltip({ content, children, className }: InfoTooltipProps) {
    return (
        <Tooltip>
            <TooltipTrigger className={cn("cursor-help flex items-center opacity-50 hover:opacity-100 transition-opacity", className)}>
                {children || <HugeiconsIcon icon={HelpCircleIcon} size={14} />}
            </TooltipTrigger>
            <TooltipContent className="max-w-[220px] text-[10px] leading-tight p-3 bg-popover text-popover-foreground border border-border shadow-md">
                {content.split('\n\n').map((part, i) => (
                    <div key={i} className={i > 0 ? "mt-2" : ""}>
                        {part.split(/(\*\*.*?\*\*)/).map((segment, j) => {
                            if (segment.startsWith('**') && segment.endsWith('**')) {
                                return <span key={j} className="font-semibold text-foreground">{segment.slice(2, -2)}</span>
                            }
                            return segment
                        })}
                    </div>
                ))}
            </TooltipContent>
        </Tooltip>
    )
}
