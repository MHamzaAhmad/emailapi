import { useState } from "react";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

interface CodeWindowProps {
    className?: string;
    tabs: {
        label: string;
        value: string;
        language: "typescript" | "bash" | "go" | "json";
    }[];
}

export function CodeWindow({ className, tabs }: CodeWindowProps) {
    const [activeTab, setActiveTab] = useState(tabs[0].value);

    return (
        <TooltipProvider delayDuration={100}>
            <div
                className={cn(
                    "w-full rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden text-left",
                    className
                )}
            >
                {/* Header - Aligned with Dashboard Table Header */}
                <div className="flex h-10 items-center justify-between border-b border-border/60 bg-muted/30 px-3">
                    <div className="flex items-center gap-1.5">
                        {tabs.map((tab) => (
                            <button
                                key={tab.value}
                                onClick={() => setActiveTab(tab.value)}
                                className={cn(
                                    "px-2.5 h-6 text-[9px] font-bold uppercase tracking-widest transition-all rounded-md border select-none",
                                    activeTab === tab.value
                                        ? "border-solid border-primary bg-primary/10 text-primary shadow-sm"
                                        : "border-dashed border-border/60 text-muted-foreground hover:border-border hover:bg-muted/50 hover:text-foreground bg-transparent"
                                )}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </div>
                </div>

                {/* Code Content Area */}
                <div className="relative bg-background/50 p-6">
                    <div className="font-mono text-xs leading-relaxed overflow-x-auto whitespace-pre">
                        <pre className="text-muted-foreground/90">
                            <RichCodeRenderer language={activeTab} />
                        </pre>
                    </div>
                </div>
            </div>
        </TooltipProvider>
    );
}

// Custom High-Fidelity Tokenizer for the Landing Page demo
function RichCodeRenderer({ language }: { language: string }) {
    if (language === "ts" || language === "typescript") {
        return (
            <span className="text-foreground/80">
                <Keyword>import</Keyword> <Punctuation>{"{"}</Punctuation> <HoverFunction name="createClient">createClient</HoverFunction> <Punctuation>{"}"}</Punctuation> <Keyword>from</Keyword> <String>'simpleemailapi'</String>
                {"\n\n"}
                <Keyword>const</Keyword> client <Operator>=</Operator> <HoverFunction name="createClient">createClient</HoverFunction><Punctuation>({"{"}</Punctuation> apiKey<Punctuation>:</Punctuation> <String>'sea_live_...'</String> <Punctuation>{"}"})</Punctuation>
                {"\n\n"}
                <Comment>// Send an email</Comment>
                {"\n"}
                <Keyword>await</Keyword> client<Punctuation>.</Punctuation><HoverFunction name="send">send</HoverFunction><Punctuation>({"{"}</Punctuation>
                {"\n"}
                {"  "}from<Punctuation>:</Punctuation> <String>'hello@yourapp.com'</String><Punctuation>,</Punctuation>
                {"\n"}
                {"  "}to<Punctuation>:</Punctuation> <Punctuation>[</Punctuation><String>'user@example.com'</String><Punctuation>],</Punctuation>
                {"\n"}
                {"  "}subject<Punctuation>:</Punctuation> <String>'Welcome!'</String><Punctuation>,</Punctuation>
                {"\n"}
                {"  "}body<Punctuation>:</Punctuation> <String>'Thanks for signing up.'</String>
                {"\n"}
                <Punctuation>{"}"})</Punctuation>
                {"\n\n"}
                <Comment>// Listen for replies</Comment>
                {"\n"}
                client<Punctuation>.</Punctuation><HoverFunction name="onReceive">onReceive</HoverFunction><Punctuation>({"{"}</Punctuation>
                {"\n"}
                {"  "}onReplied<Punctuation>:</Punctuation> <Punctuation>(</Punctuation>reply<Punctuation>)</Punctuation> <Operator>{'=>'}</Operator> console<Punctuation>.</Punctuation><Method>log</Method><Punctuation>(</Punctuation><String>'New reply:'</String><Punctuation>,</Punctuation> reply<Punctuation>.</Punctuation>body<Punctuation>)</Punctuation>
                {"\n"}
                <Punctuation>{"}"})</Punctuation>
            </span>
        )
    }

    if (language === "bash" || language === "curl") {
        return (
            <span className="text-foreground/80">
                <Keyword>curl</Keyword> <Operator>-</Operator><Keyword>X</Keyword> POST https://api.simpleemailapi.dev/v1.EmailService/Send <Operator>{"\\"}</Operator>
                {"\n"}
                {"  "}<Operator>-</Operator><Keyword>H</Keyword> <String>"Authorization: Bearer sea_live_..."</String> <Operator>{"\\"}</Operator>
                {"\n"}
                {"  "}<Operator>-</Operator><Keyword>d</Keyword> <String>'{`{`}'</String>
                {"\n"}
                {"    "}<String>"from"</String><Punctuation>:</Punctuation> <String>"updates@app.com"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"to"</String><Punctuation>:</Punctuation> <Punctuation>[</Punctuation><String>"user@example.com"</String><Punctuation>]</Punctuation><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"subject"</String><Punctuation>:</Punctuation> <String>"Welcome!"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"body"</String><Punctuation>:</Punctuation> <String>"Thanks for signing up."</String>
                {"\n"}
                {"  "}<String>'{`}`}'</String>
            </span>
        )
    }

    if (language === "go") {
        return (
            <span className="text-foreground/80">
                <Keyword>package</Keyword> main
                {"\n\n"}
                <Keyword>import</Keyword> <String>"github.com/simpleemailapi/go-sdk"</String>
                {"\n\n"}
                <Keyword>func</Keyword> <Method>main</Method><Punctuation>()</Punctuation> <Punctuation>{"{"}</Punctuation>
                {"\n"}
                {"  "}client <Operator>:=</Operator> simpleemailapi<Punctuation>.</Punctuation><Method>NewClient</Method><Punctuation>(</Punctuation><String>"sea_live_..."</String><Punctuation>)</Punctuation>
                {"\n\n"}
                {"  "}<Comment>// Send Email</Comment>
                {"\n"}
                {"  "}client<Punctuation>.</Punctuation><Method>Send</Method><Punctuation>(</Punctuation><Operator>&</Operator>simpleemailapi<Punctuation>.</Punctuation>Message<Punctuation>{"{"}</Punctuation>
                {"\n"}
                {"    "}From<Punctuation>:</Punctuation>    <String>"updates@app.com"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}To<Punctuation>:</Punctuation>      <Punctuation>[]</Punctuation><Keyword>string</Keyword><Punctuation>{"{"}</Punctuation><String>"user@example.com"</String><Punctuation>{"}"}</Punctuation><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}Subject<Punctuation>:</Punctuation> <String>"Welcome!"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}Body<Punctuation>:</Punctuation>    <String>"Thanks for signing up."</String><Punctuation>,</Punctuation>
                {"\n"}
                {"  "}<Punctuation>{"}"}</Punctuation><Punctuation>)</Punctuation>
                {"\n"}
                <Punctuation>{"}"}</Punctuation>
            </span>
        )
    }

    if (language === "json") {
        return (
            <span className="text-foreground/80">
                <Punctuation>{"{"}</Punctuation>
                {"\n"}
                {"  "}<String>"id"</String><Punctuation>:</Punctuation> <String>"evt_29f8a7..."</String><Punctuation>,</Punctuation>
                {"\n"}
                {"  "}<String>"type"</String><Punctuation>:</Punctuation> <String>"email.replied"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"  "}<String>"timestamp"</String><Punctuation>:</Punctuation> <String>"2024-03-20T10:00:00Z"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"  "}<String>"payload"</String><Punctuation>:</Punctuation> <Punctuation>{"{"}</Punctuation>
                {"\n"}
                {"    "}<String>"from"</String><Punctuation>:</Punctuation> <String>"user@gmail.com"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"to"</String><Punctuation>:</Punctuation> <Punctuation>[</Punctuation><String>"support@yourdomain.com"</String><Punctuation>]</Punctuation><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"subject"</String><Punctuation>:</Punctuation> <String>"Re: Welcome to our App"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"body"</String><Punctuation>:</Punctuation> <String>"Thanks for the email! I have a question..."</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"in_reply_to"</String><Punctuation>:</Punctuation> <String>"msg_original_123"</String><Punctuation>,</Punctuation>
                {"\n"}
                {"    "}<String>"thread_id"</String><Punctuation>:</Punctuation> <String>"thread_abc_123"</String>
                {"\n"}
                {"  "}<Punctuation>{"}"}</Punctuation>
                {"\n"}
                <Punctuation>{"}"}</Punctuation>
            </span>
        )
    }
}

// Token Components for Consistent Colors - Aligned with "Serious SaaS" Theme
const Keyword = ({ children }: { children: React.ReactNode }) => <span className="text-primary font-bold">{children}</span>;
const String = ({ children }: { children: React.ReactNode }) => <span className="text-foreground/90 font-medium">{children}</span>;
const Punctuation = ({ children }: { children: React.ReactNode }) => <span className="text-muted-foreground/60">{children}</span>;
const Operator = ({ children }: { children: React.ReactNode }) => <span className="text-muted-foreground">{children}</span>;
const Method = ({ children }: { children: React.ReactNode }) => <span className="text-foreground font-semibold">{children}</span>;
const Comment = ({ children }: { children: React.ReactNode }) => <span className="text-muted-foreground/50 italic text-[10px] uppercase tracking-wider font-bold">{children}</span>;

// Interactive Hover Component
function HoverFunction({ name, children }: { name: string, children: React.ReactNode }) {
    let signature = "";
    let description = "";

    if (name === "createClient") {
        signature = "function createClient(options: ClientOptions): Client";
        description = "Initializes a new EmailAPI client with your project's API key. Returns a client instance for sending mails and listening to events.";
    } else if (name === "send") {
        signature = "(method) Client.send(message: MessageOptions): Promise<SendResult>";
        description = "Sends a transactional email. Supports HTML content, attachments, and custom headers. Returns a promise that resolves when the email is queued.";
    } else if (name === "onReceive") {
        signature = "(method) Client.onReceive(handlers: EventHandlers): void";
        description = "Registers real-time event listeners for inbound emails. Uses a persistent trailing connection to stream events with low latency.";
    }

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <span className="text-primary hover:text-primary/80 cursor-help border-b border-dashed border-primary/30 hover:border-primary transition-colors">
                    {children}
                </span>
            </TooltipTrigger>
            <TooltipContent side="top" align="start" className="bg-popover border border-border text-popover-foreground p-0 overflow-hidden shadow-xl max-w-sm rounded-md">
                <div className="bg-muted/30 px-3 py-2 border-b border-border/40 text-xs font-mono text-primary font-medium">
                    {signature}
                </div>
                <div className="px-3 py-2 text-xs text-muted-foreground leading-relaxed">
                    {description}
                </div>
            </TooltipContent>
        </Tooltip>
    )
}
