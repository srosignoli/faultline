import { useState } from "react";
import type { Simulator } from "../api/client";

interface Props {
  simulators: Simulator[];
  onDelete: (name: string) => Promise<void>;
}

const mutatorColors: Record<string, string> = {
  spike: "bg-purple-100 text-purple-800",
  wave: "bg-blue-100 text-blue-800",
  jitter: "bg-orange-100 text-orange-800",
  trend: "bg-green-100 text-green-800",
};

function mutatorBadgeClass(type: string): string {
  return mutatorColors[type] ?? "bg-gray-100 text-gray-800";
}

export default function SimulatorList({ simulators, onDelete }: Props) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [deleting, setDeleting] = useState<Set<string>>(new Set());

  function toggleExpand(name: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }

  async function handleDelete(name: string) {
    setDeleting((prev) => new Set(prev).add(name));
    try {
      await onDelete(name);
    } finally {
      setDeleting((prev) => {
        const next = new Set(prev);
        next.delete(name);
        return next;
      });
    }
  }

  if (simulators.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-gray-400">
        <svg
          className="w-12 h-12 mb-3"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M5 12h14M12 5l7 7-7 7"
          />
        </svg>
        <p className="text-sm">No simulators running</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      {simulators.map((sim) => {
        const isExpanded = expanded.has(sim.name);
        const isDeleting = deleting.has(sim.name);

        return (
          <div key={sim.name} className="border border-gray-200 rounded-lg overflow-hidden">
            {/* Header */}
            <div
              className="flex items-center gap-3 px-4 py-3 bg-white cursor-pointer hover:bg-gray-50 select-none"
              onClick={() => toggleExpand(sim.name)}
            >
              <span className="font-semibold text-gray-900 flex-1">{sim.name}</span>
              <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">
                {sim.status ?? "running"}
              </span>
              <svg
                className={`w-4 h-4 text-gray-500 transition-transform ${isExpanded ? "rotate-180" : ""}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(sim.name);
                }}
                disabled={isDeleting}
                className="ml-2 text-xs font-medium px-3 py-1 rounded-md bg-red-50 text-red-600 hover:bg-red-100 disabled:opacity-50 transition-colors"
              >
                {isDeleting ? "Deleting…" : "Delete"}
              </button>
            </div>

            {/* Expanded rule table */}
            {isExpanded && (
              <div className="border-t border-gray-200 bg-gray-50 px-4 py-3">
                {sim.active_rules.length === 0 ? (
                  <p className="text-sm text-gray-400">No active rules</p>
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-left text-xs text-gray-500 uppercase tracking-wide">
                        <th className="pb-2 pr-4 font-medium">Target</th>
                        <th className="pb-2 pr-4 font-medium">Mutator</th>
                        <th className="pb-2 font-medium">Params</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {sim.active_rules.map((rule) => (
                        <tr key={rule.name} className="align-top">
                          <td className="py-2 pr-4">
                            <span className="font-semibold text-gray-800">{rule.name}</span>
                            <br />
                            <span className="font-mono text-xs text-gray-500">{rule.match.metric_name}</span>
                            {rule.match.labels && Object.keys(rule.match.labels).length > 0 && (
                              <div className="flex flex-wrap gap-1 mt-1">
                                {Object.entries(rule.match.labels).map(([k, v]) => (
                                  <span
                                    key={k}
                                    className="text-xs px-1.5 py-0.5 bg-gray-200 text-gray-600 rounded font-mono"
                                  >
                                    {k}={v}
                                  </span>
                                ))}
                              </div>
                            )}
                          </td>
                          <td className="py-2 pr-4">
                            <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${mutatorBadgeClass(rule.mutator.type)}`}>
                              {rule.mutator.type}
                            </span>
                          </td>
                          <td className="py-2">
                            <div className="font-mono text-xs text-gray-700 space-y-0.5">
                              {Object.entries(rule.mutator.params).map(([k, v]) => (
                                <div key={k}>
                                  <span className="text-gray-500">{k}:</span> {String(v)}
                                </div>
                              ))}
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
