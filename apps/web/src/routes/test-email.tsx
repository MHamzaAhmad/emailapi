import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { Send, Plus, Trash2, Mail, CheckCircle, XCircle, Reply, ExternalLink, Trash } from 'lucide-react';
import { useSendEmail, useLocalEmails } from '@/hooks';
import { SendEmailRequest } from '@/types';
import { webhookService } from '@/services/webhookService';

export const Route = createFileRoute('/test-email')({
    component: TestEmailPage,
});

function TestEmailPage() {
    const [formData, setFormData] = useState<SendEmailRequest>({
        from: '',
        to: [''],
        subject: '',
        html: '',
        body: '',
        inReplyTo: '',
    });
    const [useHtml, setUseHtml] = useState(false);
    const [successId, setSuccessId] = useState<string | null>(null);
    const [portalLoading, setPortalLoading] = useState(false);

    const sendEmailMutation = useSendEmail();
    const { emails, addEmail, clearEmails } = useLocalEmails();

    const handleAddField = (field: 'to' | 'cc' | 'bcc') => {
        setFormData((prev) => ({
            ...prev,
            [field]: [...(prev[field] || []), ''],
        }));
    };

    const handleRemoveField = (field: 'to' | 'cc' | 'bcc', index: number) => {
        setFormData((prev) => ({
            ...prev,
            [field]: (prev[field] || []).filter((_, i) => i !== index),
        }));
    };

    const handleChangeField = (
        field: 'to' | 'cc' | 'bcc',
        index: number,
        value: string
    ) => {
        setFormData((prev) => ({
            ...prev,
            [field]: (prev[field] || []).map((item, i) => (i === index ? value : item)),
        }));
    };

    const handleReply = (email: any) => {
        setFormData({
            from: email.to[0] || '', // Reply from the recipient (us)
            to: [email.from], // Reply to the sender
            subject: email.subject.startsWith('Re:') ? email.subject : `Re: ${email.subject}`,
            body: '',
            html: '',
            inReplyTo: email.messageId, // Use messageId for threading
        });
        setUseHtml(false);
        // Scroll to top
        window.scrollTo({ top: 0, behavior: 'smooth' });
    };

    const handleOpenPortal = async () => {
        setPortalLoading(true);
        try {
            const res = await webhookService.getAppPortalAccess();
            window.open(res.url, '_blank');
        } catch (error) {
            console.error('Failed to get portal access:', error);
            alert('Failed to open webhook portal. Check console for details.');
        } finally {
            setPortalLoading(false);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSuccessId(null);

        // Filter out empty recipients
        const cleanData = {
            ...formData,
            to: formData.to.filter((e) => e),
            cc: formData.cc?.filter((e) => e),
            bcc: formData.bcc?.filter((e) => e),
            inReplyTo: formData.inReplyTo || undefined,
        };

        if (useHtml) {
            cleanData.html = cleanData.body;
            cleanData.body = undefined;
        } else {
            cleanData.html = undefined;
        }

        try {
            const res = await sendEmailMutation.mutateAsync(cleanData);
            setSuccessId(res.id);

            // Add to local store
            addEmail({
                id: res.id,
                messageId: res.message_id, // Important for replies
                from: cleanData.from,
                to: cleanData.to,
                subject: cleanData.subject,
                body: cleanData.body || cleanData.html || '',
                status: cleanData.async ? 'QUEUED' : 'SENT',
            });

        } catch (error) {
            console.error('Failed to send email:', error);
        }
    };

    return (
        <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
            <div className="max-w-6xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-8">
                {/* Left Column: Sending Form */}
                <div>
                    <div className="flex items-center gap-3 mb-8">
                        <Mail className="w-8 h-8 text-cyan-400" />
                        <h1 className="text-3xl font-bold text-white">Test Email</h1>
                    </div>

                    {successId && (
                        <div className="mb-6 bg-green-500/10 border border-green-500/50 rounded-lg p-4 flex items-center gap-3">
                            <CheckCircle className="w-5 h-5 text-green-400" />
                            <div>
                                <p className="text-green-400 font-medium">Email Sent Successfully!</p>
                                <p className="text-sm text-green-300/80">ID: {successId}</p>
                            </div>
                        </div>
                    )}

                    {sendEmailMutation.isError && (
                        <div className="mb-6 bg-red-500/10 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
                            <XCircle className="w-5 h-5 text-red-400" />
                            <div>
                                <p className="text-red-400 font-medium">Failed to send email</p>
                                <p className="text-sm text-red-300/80">
                                    {sendEmailMutation.error instanceof Error
                                        ? sendEmailMutation.error.message
                                        : 'Unknown error'}
                                </p>
                            </div>
                        </div>
                    )}

                    <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6 sticky top-6">
                        <form onSubmit={handleSubmit} className="space-y-6">
                            {/* In-Reply-To Hidden/Context */}
                            {formData.inReplyTo && (
                                <div className="bg-cyan-500/10 border border-cyan-500/30 rounded-lg p-3 flex items-center justify-between">
                                    <span className="text-sm text-cyan-300">Replying to Message-ID: {formData.inReplyTo}</span>
                                    <button
                                        type="button"
                                        onClick={() => setFormData(prev => ({ ...prev, inReplyTo: '' }))}
                                        className="text-gray-400 hover:text-white"
                                    >
                                        <XCircle className="w-4 h-4" />
                                    </button>
                                </div>
                            )}

                            {/* From */}
                            <div>
                                <label className="block text-gray-400 mb-2">From</label>
                                <input
                                    type="email"
                                    required
                                    value={formData.from}
                                    onChange={(e) =>
                                        setFormData((prev) => ({ ...prev, from: e.target.value }))
                                    }
                                    placeholder="sender@yourdomain.com"
                                    className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
                                />
                            </div>

                            {/* To Recipients */}
                            <div>
                                <label className="block text-gray-400 mb-2">To</label>
                                <div className="space-y-2">
                                    {formData.to.map((email, index) => (
                                        <div key={index} className="flex gap-2">
                                            <input
                                                type="email"
                                                required
                                                value={email}
                                                onChange={(e) => handleChangeField('to', index, e.target.value)}
                                                placeholder="recipient@example.com"
                                                className="flex-1 px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
                                            />
                                            {formData.to.length > 1 && (
                                                <button
                                                    type="button"
                                                    onClick={() => handleRemoveField('to', index)}
                                                    className="p-2 text-gray-400 hover:text-red-400 hover:bg-slate-700 rounded-lg"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            )}
                                        </div>
                                    ))}
                                </div>
                                <button
                                    type="button"
                                    onClick={() => handleAddField('to')}
                                    className="mt-2 text-sm text-cyan-400 hover:text-cyan-300 flex items-center gap-1"
                                >
                                    <Plus className="w-3 h-3" /> Add Recipient
                                </button>
                            </div>

                            {/* Subject */}
                            <div>
                                <label className="block text-gray-400 mb-2">Subject</label>
                                <input
                                    type="text"
                                    required
                                    value={formData.subject}
                                    onChange={(e) =>
                                        setFormData((prev) => ({ ...prev, subject: e.target.value }))
                                    }
                                    placeholder="Hello World"
                                    className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
                                />
                            </div>

                            {/* Content Type Toggle */}
                            <div className="flex items-center gap-4">
                                <span className={`text-sm ${!useHtml ? 'text-cyan-400 font-medium' : 'text-gray-400'}`}>Text</span>
                                <button
                                    type="button"
                                    onClick={() => setUseHtml(!useHtml)}
                                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${useHtml ? 'bg-cyan-600' : 'bg-slate-600'
                                        }`}
                                >
                                    <span
                                        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${useHtml ? 'translate-x-6' : 'translate-x-1'
                                            }`}
                                    />
                                </button>
                                <span className={`text-sm ${useHtml ? 'text-cyan-400 font-medium' : 'text-gray-400'}`}>HTML</span>
                            </div>

                            {/* Body */}
                            <div>
                                <label className="block text-gray-400 mb-2">Body ({useHtml ? 'HTML' : 'Text'})</label>
                                <textarea
                                    required
                                    rows={6}
                                    value={formData.body}
                                    onChange={(e) =>
                                        setFormData((prev) => ({ ...prev, body: e.target.value }))
                                    }
                                    placeholder={useHtml ? '<h1>Hello</h1>' : 'Hello there'}
                                    className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none font-mono"
                                />
                            </div>

                            <button
                                type="submit"
                                disabled={sendEmailMutation.isPending}
                                className="w-full py-3 bg-cyan-500 hover:bg-cyan-600 disabled:bg-slate-600 disabled:cursor-not-allowed text-white font-bold rounded-lg transition-colors flex items-center justify-center gap-2"
                            >
                                {sendEmailMutation.isPending ? (
                                    <>Sending...</>
                                ) : (
                                    <>
                                        <Send className="w-5 h-5" /> Send Email
                                    </>
                                )}
                            </button>
                        </form>
                    </div>
                </div>

                {/* Right Column: Sent Emails List & Webhooks */}
                <div>
                    <div className="flex items-center justify-between mb-8">
                        <h2 className="text-2xl font-bold text-white">Sent Emails</h2>
                        <div className="flex gap-2">
                            <button
                                onClick={handleOpenPortal}
                                disabled={portalLoading}
                                className="flex items-center gap-2 px-3 py-2 bg-slate-700 hover:bg-slate-600 text-cyan-400 rounded-lg transition-colors text-sm font-medium"
                            >
                                <ExternalLink className="w-4 h-4" />
                                {portalLoading ? 'Loading...' : 'View Webhooks'}
                            </button>
                            <button
                                onClick={clearEmails}
                                className="p-2 text-gray-400 hover:text-red-400 hover:bg-slate-700 rounded-lg transition-colors"
                                title="Clear History"
                            >
                                <Trash className="w-5 h-5" />
                            </button>
                        </div>
                    </div>

                    <div className="mb-4 text-xs text-gray-400 bg-slate-800/30 p-3 rounded-lg border border-slate-700/50">
                        Connect your inbox to see replies. Click "View Webhooks" to verify reply events.
                    </div>

                    <div className="space-y-4">
                        {emails.map((email) => (
                            <div key={email.id} className="bg-slate-800/50 border border-slate-700 rounded-xl p-4 transition-all hover:border-slate-600">
                                <div className="flex justify-between items-start mb-2">
                                    <div>
                                        <h3 className="text-white font-medium truncate max-w-xs" title={email.subject}>
                                            {email.subject || '(No Subject)'}
                                        </h3>
                                        <p className="text-sm text-gray-400 mt-1">
                                            To: {email.to.join(', ')}
                                        </p>
                                    </div>
                                    <span className={`px-2 py-1 text-xs rounded-full border ${email.status === 'SENT'
                                        ? 'bg-green-500/10 text-green-400 border-green-500/20'
                                        : 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
                                        }`}>
                                        {email.status}
                                    </span>
                                </div>

                                <div className="flex justify-between items-center mt-4 pt-3 border-t border-slate-700/50">
                                    <div className="text-xs text-gray-500 font-mono">
                                        ID: {email.id.slice(0, 8)}...
                                    </div>
                                    <button
                                        onClick={() => handleReply(email)}
                                        className="text-sm text-cyan-400 hover:text-cyan-300 flex items-center gap-1 px-3 py-1.5 hover:bg-cyan-500/10 rounded-lg transition-colors"
                                    >
                                        <Reply className="w-3 h-3" /> Reply
                                    </button>
                                </div>
                            </div>
                        ))}

                        {emails.length === 0 && (
                            <div className="text-center py-12 text-gray-500">
                                No emails sent in this session.
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
