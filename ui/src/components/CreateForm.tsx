import { useState } from "react";

interface Props {
  onSubmit: (name: string, dump: string, rules: string) => Promise<void>;
}

export default function CreateForm({ onSubmit }: Props) {
  const [name, setName] = useState("");
  const [dump, setDump] = useState("");
  const [rules, setRules] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await onSubmit(name, dump, rules);
      setName("");
      setDump("");
      setRules("");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Simulator Name
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          placeholder="my-simulator"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Metric Dump
        </label>
        <textarea
          value={dump}
          onChange={(e) => setDump(e.target.value)}
          required
          rows={8}
          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
          placeholder="# HELP http_requests_total Total HTTP requests&#10;# TYPE http_requests_total counter&#10;http_requests_total{method=&quot;GET&quot;} 1234"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Mutation Rules (YAML)
        </label>
        <textarea
          value={rules}
          onChange={(e) => setRules(e.target.value)}
          required
          rows={8}
          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
          placeholder="rules:&#10;  - name: spike-requests&#10;    match:&#10;      metric_name: http_requests_total&#10;    mutator:&#10;      type: spike&#10;      params:&#10;        multiplier: 10&#10;        duration: 30s"
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium py-2 px-4 rounded-md text-sm transition-colors"
      >
        {loading ? "Deploying…" : "Deploy Simulator"}
      </button>
    </form>
  );
}
