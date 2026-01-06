"use client";

import { useState, useEffect, useRef } from "react";

interface Attachment {
  filename: string;
  contentType: string;
  type: "url" | "base64";
  value: string;
}

interface LogEvent {
  id: string;
  timestamp: string;
  data: any;
}

export default function Home() {
  // Form State
  const [from, setFrom] = useState("sender@halftwin.com");
  const [to, setTo] = useState("hamzabuzz88@gmail.com");
  const [subject, setSubject] = useState("Test Email");
  const [body, setBody] = useState("This is a test email body.");
  const [isAsync, setIsAsync] = useState(false);
  const [attachments, setAttachments] = useState<Attachment[]>([]);

  // UI State
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const [events, setEvents] = useState<LogEvent[]>([]);

  // Refs
  const eventSourceRef = useRef<EventSource | null>(null);

  // SSE Connection
  useEffect(() => {
    const connectSSE = () => {
      const es = new EventSource("/api/email/receive");

      es.onopen = () => {
        setConnected(true);
        addEvent("System", "Connected to SSE stream");
      };

      es.onerror = () => {
        setConnected(false);
        // addEvent("System", "SSE Connection error (retrying...)");
      };

      es.onmessage = (event) => {
        try {
          // Check if data is JSON
          const data = JSON.parse(event.data);
          addEvent("Received", data);
        } catch (e) {
          addEvent("Received (Raw)", event.data);
        }

      };

      eventSourceRef.current = es;
    };

    connectSSE();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  const addEvent = (type: string, data: any) => {
    setEvents((prev) => [
      {
        id: crypto.randomUUID(),
        timestamp: new Date().toLocaleTimeString(),
        data: { type, payload: data },
      },
      ...prev,
    ]);
  };

  const handleAddAttachment = () => {
    setAttachments([
      ...attachments,
      { filename: "test.txt", contentType: "text/plain", type: "url", value: "https://example.com/test.txt" },
    ]);
  };

  const handleUpdateAttachment = (index: number, field: keyof Attachment, value: string) => {
    const newAtts = [...attachments];
    newAtts[index] = { ...newAtts[index], [field]: value };
    setAttachments(newAtts);
  };

  const handleRemoveAttachment = (index: number) => {
    setAttachments(attachments.filter((_, i) => i !== index));
  };


  const handleSend = async () => {
    setLoading(true);
    setResponse(null);

    // Determine Endpoint
    const hasAttachments = attachments.length > 0;
    const endpoint = hasAttachments ? "/api/email/attachments" : "/api/email/send";

    const payload: any = {
      from,
      to: [to], // Backend expects an array of email addresses
      subject,
      body,
      async: isAsync,
    };

    if (hasAttachments) {
      payload.attachments = attachments.map(att => ({
        filename: att.filename,
        contentType: att.contentType,
        [att.type]: att.value
      }))
    }

    try {
      const res = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const data = await res.json();
      setResponse(JSON.stringify(data, null, 2));

      if (res.ok) {
        addEvent("Sent", { endpoint, status: res.status });
      } else {
        addEvent("Error", { endpoint, status: res.status, error: data });
      }

    } catch (error) {
      setResponse(`Error: ${error}`);
      addEvent("Error", { error: String(error) });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-black text-zinc-900 dark:text-zinc-100 p-8 font-sans">
      <div className="max-w-6xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-8">

        {/* Left Column: Send Form */}
        <div className="space-y-6">
          <h1 className="text-2xl font-bold">Send Email</h1>

          <div className="space-y-4 bg-white dark:bg-zinc-900 p-6 rounded-lg border border-zinc-200 dark:border-zinc-800 shadow-sm">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1">From</label>
                <input
                  className="w-full bg-zinc-50 dark:bg-zinc-800 border-none rounded px-3 py-2 text-sm"
                  value={from}
                  onChange={(e) => setFrom(e.target.value)}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">To</label>
                <input
                  className="w-full bg-zinc-50 dark:bg-zinc-800 border-none rounded px-3 py-2 text-sm"
                  value={to}
                  onChange={(e) => setTo(e.target.value)}
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Subject</label>
              <input
                className="w-full bg-zinc-50 dark:bg-zinc-800 border-none rounded px-3 py-2 text-sm"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Body</label>
              <textarea
                className="w-full bg-zinc-50 dark:bg-zinc-800 border-none rounded px-3 py-2 text-sm font-mono h-32"
                value={body}
                onChange={(e) => setBody(e.target.value)}
              />
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="async"
                checked={isAsync}
                onChange={e => setIsAsync(e.target.checked)}
                className="rounded border-zinc-300 dark:border-zinc-700"
              />
              <label htmlFor="async" className="text-sm font-medium">Send Asynchronously</label>
            </div>

            {/* Attachments Section */}
            <div>
              <div className="flex justify-between items-center mb-2">
                <label className="block text-sm font-medium">Attachments</label>
                <div className="flex gap-2">
                  <label className="text-xs bg-zinc-200 dark:bg-zinc-800 px-2 py-1 rounded hover:bg-zinc-300 dark:hover:bg-zinc-700 cursor-pointer">
                    Upload File
                    <input
                      type="file"
                      className="hidden"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (!file) return;

                        const reader = new FileReader();
                        reader.onload = (e) => {
                          const result = e.target?.result as string;
                          // Result is "data:contentType;base64,....", we just want the base64 part for the API usually, 
                          // but let's check what the API expects. 
                          // The API wrapper usually expects just the base64 string, so we strip the prefix.
                          const base64 = result.split(',')[1];

                          setAttachments(prev => [...prev, {
                            filename: file.name,
                            contentType: file.type || "application/octet-stream",
                            type: "base64",
                            value: base64
                          }]);
                        };
                        reader.readAsDataURL(file);

                        // Reset input
                        e.target.value = "";
                      }}
                    />
                  </label>
                  <button onClick={handleAddAttachment} className="text-xs bg-zinc-200 dark:bg-zinc-800 px-2 py-1 rounded hover:bg-zinc-300 dark:hover:bg-zinc-700">
                    + Manual
                  </button>
                </div>
              </div>
              <div className="space-y-3">
                {attachments.map((att, i) => (
                  <div key={i} className="p-3 bg-zinc-50 dark:bg-zinc-800/50 rounded border border-zinc-200 dark:border-zinc-800 text-xs space-y-2">
                    <div className="flex gap-2">
                      <input
                        placeholder="Filename"
                        className="flex-1 bg-transparent border-b border-zinc-300 dark:border-zinc-700 p-1"
                        value={att.filename}
                        onChange={e => handleUpdateAttachment(i, "filename", e.target.value)}
                      />
                      <input
                        placeholder="Content-Type"
                        className="w-1/3 bg-transparent border-b border-zinc-300 dark:border-zinc-700 p-1"
                        value={att.contentType}
                        onChange={e => handleUpdateAttachment(i, "contentType", e.target.value)}
                      />
                      <button onClick={() => handleRemoveAttachment(i)} className="text-red-500 font-bold px-2">×</button>
                    </div>
                    <div className="flex gap-2 items-center">
                      <select
                        className="bg-transparent border border-zinc-300 dark:border-zinc-700 rounded p-1"
                        value={att.type}
                        onChange={e => handleUpdateAttachment(i, "type", e.target.value as "url" | "base64")}
                      >
                        <option value="url">URL</option>
                        <option value="base64">Base64</option>
                      </select>
                      <input
                        placeholder={att.type === "url" ? "https://..." : "Base64 Content..."}
                        className="flex-1 bg-transparent border-b border-zinc-300 dark:border-zinc-700 p-1"
                        value={att.value}
                        onChange={e => handleUpdateAttachment(i, "value", e.target.value)}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <button
              onClick={handleSend}
              disabled={loading}
              className="w-full bg-blue-600 hover:bg-blue-500 text-white font-medium py-2 px-4 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? "Sending..." : "Send Email"}
            </button>

            {response && (
              <div className="mt-4 p-4 bg-zinc-100 dark:bg-zinc-950 rounded border border-zinc-200 dark:border-zinc-800 overflow-x-auto">
                <pre className="text-xs font-mono">{response}</pre>
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Events */}
        <div className="space-y-6 flex flex-col h-[calc(100vh-6rem)]">
          <div className="flex justify-between items-center">
            <h2 className="text-2xl font-bold flex items-center gap-2">
              Events
              <span className={`w-3 h-3 rounded-full ${connected ? "bg-green-500" : "bg-red-500 animate-pulse"}`} />
            </h2>
            <button onClick={() => setEvents([])} className="text-xs text-zinc-500 hover:text-zinc-300">Clear</button>
          </div>

          <div className="flex-1 bg-zinc-900 rounded-lg border border-zinc-800 p-4 overflow-y-auto font-mono text-sm shadow-inner">
            {events.length === 0 && (
              <div className="text-zinc-600 text-center mt-10">No events received yet...</div>
            )}
            {events.map((ev) => (
              <div key={ev.id} className="mb-4 pb-4 border-b border-zinc-800 last:border-0">
                <div className="flex justify-between text-xs text-zinc-500 mb-1">
                  <span className="font-bold text-blue-400">[{ev.data.type}]</span>
                  <span>{ev.timestamp}</span>
                </div>
                <pre className="text-zinc-300 whitespace-pre-wrap break-all text-xs">
                  {JSON.stringify(ev.data.payload, null, 2)}
                </pre>
              </div>
            ))}
          </div>
        </div>

      </div>
    </div>
  );
}
