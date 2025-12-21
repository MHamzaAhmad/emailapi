import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { Send, Plus, Trash2, Mail, CheckCircle, XCircle } from 'lucide-react';
import { useSendEmail } from '@/hooks';
import { SendEmailRequest } from '@/types';

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
    });
    const [useHtml, setUseHtml] = useState(false);
    const [successId, setSuccessId] = useState<string | null>(null);

    const sendEmailMutation = useSendEmail();

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

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSuccessId(null);

        // Filter out empty recipients
        const cleanData = {
            ...formData,
            to: formData.to.filter((e) => e),
            cc: formData.cc?.filter((e) => e),
            bcc: formData.bcc?.filter((e) => e),
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
        } catch (error) {
            console.error('Failed to send email:', error);
        }
    };

    return (
        <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
            <div className="max-w-4xl mx-auto">
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

                <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
                    <form onSubmit={handleSubmit} className="space-y-6">
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
        </div>
    );
}
