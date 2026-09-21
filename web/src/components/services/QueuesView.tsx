import React, { useState, useEffect } from 'react';
import { Send, Plus, Trash2, RefreshCw, AlertCircle, MessageSquare, Inbox } from 'lucide-react';

interface Queue {
  id: string;
  name: string;
  consumerAppId: string;
  maxRetries: number;
  messageBacklog: number;
  createdAt: string;
}

interface QueueMessage {
  id: string;
  queueId: string;
  body: string;
  attempts: number;
  createdAt: string;
}

export function QueuesView({ initialSelectedId }: { initialSelectedId?: string } = {}) {
  const [queues, setQueues] = useState<Queue[]>([]);
  const [selectedQueue, setSelectedQueue] = useState<Queue | null>(null);
  const [messages, setMessages] = useState<QueueMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState('');
  const [consumerAppId, setConsumerAppId] = useState('');
  const [maxRetries, setMaxRetries] = useState(3);

  const [showMessageModal, setShowMessageModal] = useState(false);
  const [messageBody, setMessageBody] = useState('');
  const [sending, setSending] = useState(false);

  const fetchQueues = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/queues');
      if (!res.ok) throw new Error('Failed to fetch queues');
      const data = await res.json();
      const list = Array.isArray(data) ? data : [];
      setQueues(list);
      if (initialSelectedId) {
        const found = list.find(q => q.id === initialSelectedId || q.name === initialSelectedId);
        if (found) {
          setSelectedQueue(found);
          return;
        }
      }
      if (list.length > 0 && !selectedQueue) {
        setSelectedQueue(list[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialSelectedId && queues.length > 0) {
      const found = queues.find(q => q.id === initialSelectedId || q.name === initialSelectedId);
      if (found) {
        setSelectedQueue(found);
      }
    }
  }, [initialSelectedId, queues]);

  const fetchMessages = async (queueId: string) => {
    try {
      const res = await fetch(`/api/v1/queues/${queueId}/messages`);
      if (!res.ok) throw new Error('Failed to fetch messages');
      const data = await res.json();
      setMessages(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchQueues();
  }, []);

  useEffect(() => {
    if (selectedQueue) {
      fetchMessages(selectedQueue.id);
    } else {
      setMessages([]);
    }
  }, [selectedQueue]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    try {
      const res = await fetch('/api/v1/queues', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          consumer_app_id: consumerAppId.trim(),
          max_retries: Number(maxRetries) || 3,
        }),
      });
      if (!res.ok) throw new Error('Failed to create queue');
      const created = await res.json();
      setShowCreateModal(false);
      setName('');
      setConsumerAppId('');
      setMaxRetries(3);
      await fetchQueues();
      setSelectedQueue(created);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this queue?')) return;
    try {
      const res = await fetch(`/api/v1/queues/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete queue');
      if (selectedQueue?.id === id) {
        setSelectedQueue(null);
      }
      await fetchQueues();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedQueue || !messageBody.trim()) return;
    setSending(true);
    try {
      const res = await fetch(`/api/v1/queues/${selectedQueue.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body: messageBody.trim() }),
      });
      if (!res.ok) throw new Error('Failed to send message');
      setShowMessageModal(false);
      setMessageBody('');
      await fetchMessages(selectedQueue.id);
      await fetchQueues();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">Queues</h2>
            <span className="text-xs bg-emerald-950 text-emerald-400 border border-emerald-800 px-2.5 py-0.5 rounded-full font-mono font-medium">
              celld service: Yes
            </span>
          </div>
          <p className="text-sm text-zinc-400 mt-1">
            Asynchronous event and task queues with guaranteed delivery and automatic worker retries.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
        >
          <Plus className="w-4 h-4" />
          Create Queue
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl border border-red-900/50 bg-red-950/20 text-red-400 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {/* Main Grid: Left = Queue List, Right = Messages Inspector */}
      <div className="grid grid-cols-12 gap-6 min-h-[500px]">
        {/* Left Column: Queues List */}
        <div className="col-span-5 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-4 flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800">
              <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Active Queues</span>
              <button onClick={fetchQueues} className="text-zinc-500 hover:text-zinc-300 transition">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>

            {queues.length === 0 ? (
              <div className="text-center py-10 text-zinc-500 text-sm">
                No message queues created yet.
              </div>
            ) : (
              <div className="space-y-2 overflow-y-auto max-h-[480px]">
                {queues.map(q => (
                  <div
                    key={q.id}
                    onClick={() => setSelectedQueue(q)}
                    className={`p-3.5 rounded-xl cursor-pointer border transition space-y-2 ${
                      selectedQueue?.id === q.id
                        ? 'bg-zinc-800/90 border-emerald-500/50 text-white'
                        : 'border-transparent hover:bg-zinc-800/40 text-zinc-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Inbox className={`w-4 h-4 ${selectedQueue?.id === q.id ? 'text-emerald-400' : 'text-zinc-500'}`} />
                        <span className="font-semibold text-sm">{q.name}</span>
                      </div>
                      <span className="text-xs font-mono bg-zinc-800 px-2 py-0.5 rounded text-emerald-400">
                        {q.messageBacklog} backlog
                      </span>
                    </div>

                    <div className="flex items-center justify-between text-xs text-zinc-400">
                      <span>Retries: {q.maxRetries}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(q.id);
                        }}
                        className="p-1 hover:text-red-400 text-zinc-500 transition"
                        title="Delete Queue"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Queue Messages */}
        <div className="col-span-7 border border-zinc-800/80 bg-zinc-900/30 rounded-2xl p-6 flex flex-col justify-between">
          {selectedQueue ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-bold">{selectedQueue.name}</h3>
                    <span className="text-xs font-mono text-zinc-500">ID: {selectedQueue.id}</span>
                  </div>
                  <p className="text-xs text-zinc-400 mt-0.5">Consumer Worker: <span className="font-mono text-zinc-300">{selectedQueue.consumerAppId || 'Direct Poll'}</span></p>
                </div>
                <button
                  onClick={() => setShowMessageModal(true)}
                  className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-xs font-semibold transition"
                >
                  <Send className="w-3.5 h-3.5" />
                  Publish Message
                </button>
              </div>

              <div className="flex items-center justify-between text-xs text-zinc-400 font-semibold uppercase tracking-wider">
                <div className="flex items-center gap-2">
                  <MessageSquare className="w-4 h-4 text-emerald-400" />
                  <span>Message Queue Backlog ({messages.length})</span>
                </div>
                <button onClick={() => fetchMessages(selectedQueue.id)} className="text-zinc-500 hover:text-zinc-300">
                  <RefreshCw className="w-3 h-3" />
                </button>
              </div>

              <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950/50">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-zinc-800 bg-zinc-900/50 text-zinc-400 font-semibold">
                    <tr>
                      <th className="p-3">Message ID</th>
                      <th className="p-3">Payload Preview</th>
                      <th className="p-3">Attempts</th>
                      <th className="p-3">Received At</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/60 font-mono">
                    {messages.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="p-8 text-center text-zinc-500 font-sans">
                          No messages in queue backlog. Click "Publish Message" to enqueue a task.
                        </td>
                      </tr>
                    ) : (
                      messages.map(msg => (
                        <tr key={msg.id} className="hover:bg-zinc-800/30 transition">
                          <td className="p-3 text-zinc-400 font-mono text-[11px]">
                            {msg.id.slice(0, 8)}...
                          </td>
                          <td className="p-3 text-zinc-200 font-sans max-w-[240px] truncate">
                            {msg.body}
                          </td>
                          <td className="p-3 text-zinc-400">
                            {msg.attempts}
                          </td>
                          <td className="p-3 text-zinc-500 text-[11px] font-sans">
                            {new Date(msg.createdAt).toLocaleTimeString()}
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-center py-20 text-zinc-500">
              <Inbox className="w-12 h-12 stroke-1 text-zinc-700 mb-3" />
              <p className="text-sm">Select a queue to inspect messages and backlog</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Queue Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Create Message Queue</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Queue Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. notifications, webhook-dispatch"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-sm focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Consumer Worker ID / Subdomain (optional)
                </label>
                <input
                  type="text"
                  placeholder="e.g. email-sender"
                  value={consumerAppId}
                  onChange={(e) => setConsumerAppId(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Max Retries
                </label>
                <input
                  type="number"
                  min={1}
                  max={10}
                  value={maxRetries}
                  onChange={(e) => setMaxRetries(parseInt(e.target.value) || 3)}
                  className="w-full px-3 py-2 rounded-xl bg-zinc-950 border border-zinc-800 text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-zinc-950 text-sm font-semibold transition"
                >
                  Create Queue
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Publish Message Modal */}
      {showMessageModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-lg bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold">Publish Message to Queue</h3>
            <form onSubmit={handleSendMessage} className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider block mb-1">
                  Message Payload (JSON or text)
                </label>
                <textarea
                  rows={5}
                  required
                  placeholder='{"event": "order_created", "orderId": 1234, "total": 99.95}'
                  value={messageBody}
                  onChange={(e) => setMessageBody(e.target.value)}
                  className="w-full p-3 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-xs focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowMessageModal(false)}
                  className="px-4 py-2 rounded-lg text-sm text-zinc-400 hover:text-white transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={sending}
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-zinc-950 text-sm font-semibold transition flex items-center gap-2"
                >
                  <Send className="w-3.5 h-3.5" />
                  {sending ? 'Publishing...' : 'Publish Message'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
