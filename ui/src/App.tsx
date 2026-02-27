import { useEffect, useState } from "react";
import { getSimulators, createSimulator, deleteSimulator } from "./api/client";
import type { Simulator } from "./api/client";
import CreateForm from "./components/CreateForm";
import SimulatorList from "./components/SimulatorList";

export default function App() {
  const [simulators, setSimulators] = useState<Simulator[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function loadSimulators() {
    setIsLoading(true);
    try {
      const data = await getSimulators();
      setSimulators(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load simulators");
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    loadSimulators();
  }, []);

  async function handleCreate(name: string, dump: string, rules: string) {
    try {
      await createSimulator({ name, dump_payload: dump, rules_payload: rules });
      await loadSimulators();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to create simulator";
      setError(msg);
      throw err;
    }
  }

  async function handleDelete(name: string) {
    try {
      await deleteSimulator(name);
      await loadSimulators();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete simulator");
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* NavBar */}
      <header className="bg-indigo-700 text-white px-6 py-4 shadow">
        <h1 className="text-xl font-bold tracking-tight">MetricForge Control Plane</h1>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-6 flex flex-col gap-4">
        {/* Error banner */}
        {error !== null && (
          <div className="flex items-center justify-between bg-red-50 border border-red-200 text-red-700 rounded-lg px-4 py-3 text-sm">
            <span>{error}</span>
            <button
              onClick={() => setError(null)}
              className="ml-4 text-red-500 hover:text-red-700 font-medium"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Two-column layout */}
        <div className="flex gap-6 items-start">
          {/* Left: CreateForm ~40% */}
          <div className="w-2/5 bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
            <h2 className="text-base font-semibold text-gray-800 mb-4">Deploy Simulator</h2>
            <CreateForm onSubmit={handleCreate} />
          </div>

          {/* Right: SimulatorList ~60% */}
          <div className="flex-1 bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold text-gray-800">Running Simulators</h2>
              <button
                onClick={loadSimulators}
                disabled={isLoading}
                className="text-xs text-indigo-600 hover:text-indigo-800 disabled:opacity-50 font-medium"
              >
                {isLoading ? "Refreshing…" : "Refresh"}
              </button>
            </div>
            <SimulatorList simulators={simulators} onDelete={handleDelete} />
          </div>
        </div>
      </main>
    </div>
  );
}
