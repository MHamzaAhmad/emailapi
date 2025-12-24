import { useState } from "react";
import { cn } from "@/lib/utils";
import { HugeiconsIcon } from "@hugeicons/react";
import { Copy01Icon, CheckmarkCircle01Icon } from "@hugeicons/core-free-icons";

interface CodeWindowProps {
    className?: string;
    tabs: {
        label: string;
        value: string;
        language: "typescript" | "bash" | "go" | "json";
        content: string;
    }[];
}

export function CodeWindow({ className, tabs }: CodeWindowProps) {
    const [activeTab, setActiveTab] = useState(tabs[0].value);
    const [copied, setCopied] = useState(false);

    const activeContent = tabs.find((t) => t.value === activeTab)?.content || "";

    const handleCopy = () => {
        navigator.clipboard.writeText(activeContent);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    return (
        <div
            className={cn(
                "relative rounded-lg border border-border bg-zinc-950 shadow-sm overflow-hidden group text-left",
                className
            )}
        >
            {/* Header / Tabs */}
            <div className="flex items-center justify-between border-b border-white/5 bg-white/5 px-2">
                <div className="flex items-center gap-1">
                    {tabs.map((tab) => (
                        <button
                            key={tab.value}
                            onClick={() => setActiveTab(tab.value)}
                            className={cn(
                                "relative px-3 py-2.5 text-xs font-medium transition-colors hover:text-white outline-none",
                                activeTab === tab.value
                                    ? "text-white"
                                    : "text-zinc-500"
                            )}
                        >
                            {tab.label}
                            {activeTab === tab.value && (
                                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
                            )}
                        </button>
                    ))}
                </div>
                <div className="flex items-center gap-2 pr-2">
                    <button
                        onClick={handleCopy}
                        className="flex h-6 w-6 items-center justify-center rounded-md text-zinc-500 transition-colors hover:bg-white/10 hover:text-white"
                        aria-label="Copy code"
                    >
                        <HugeiconsIcon
                            icon={copied ? CheckmarkCircle01Icon : Copy01Icon}
                            size={14}
                        />
                    </button>
                </div>
            </div>

            {/* Code Area */}
            <div className="relative p-6 font-mono text-xs leading-relaxed overflow-x-auto">
                {/* Syntax Highlighting Simulation - Dark Mode Forced */}
                <pre className="text-zinc-300">
                    <CodeRenderer content={activeContent} />
                </pre>
            </div>
        </div>
    );
}

function CodeRenderer({ content }: { content: string; language?: string }) {
    // Very basic syntax highlighting for the specific demo snippets
    // Colors tuned for Zinc-950 background
    const tokens = content.split(/(\s+|[(){}[\]:,="'`])/);

    return (
        <>
            {tokens.map((token, i) => {
                let color = "text-zinc-300"; // default

                // Keywords
                if (["const", "import", "from", "await", "function", "return", "package", "func", "type", "struct", "if", "else", "var"].includes(token)) color = "text-violet-400";
                // Booleans / Null / Numbers
                else if (["true", "false", "null", "nil"].includes(token) || !isNaN(Number(token))) color = "text-orange-400";
                // Strings
                else if (token.startsWith('"') || token.startsWith("'") || token.startsWith("`")) color = "text-emerald-400";
                // Methods / Functions
                else if (token.match(/^[a-zA-Z]+\(/)) color = "text-blue-400"; // simplistic
                // Types
                else if (["Email", "Client"].includes(token)) color = "text-yellow-400";
                // Comments
                else if (token.startsWith("//")) color = "text-zinc-500";

                // Specific manual overrides for known keys
                if (['"from"', '"to"', '"subject"', '"body"'].includes(token)) color = "text-sky-300";

                return <span key={i} className={color}>{token}</span>;
            })}
        </>
    )
}
