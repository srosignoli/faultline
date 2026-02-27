export interface Mutator {
  type: string;
  params: Record<string, unknown>;
}

export interface RuleMatch {
  metric_name: string;
  labels?: Record<string, string>;
}

export interface Rule {
  name: string;
  match: RuleMatch;
  mutator: Mutator;
}

export interface Simulator {
  name: string;
  status?: string;
  active_rules: Rule[];
}

export interface CreateSimulatorRequest {
  name: string;
  dump_payload: string;
  rules_payload: string;
}

const BASE = "/api";

async function checkResponse(res: Response): Promise<void> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`);
  }
}

export async function getSimulators(): Promise<Simulator[]> {
  const res = await fetch(`${BASE}/simulators`);
  await checkResponse(res);
  return res.json() as Promise<Simulator[]>;
}

export async function createSimulator(req: CreateSimulatorRequest): Promise<void> {
  const res = await fetch(`${BASE}/simulators`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  await checkResponse(res);
}

export async function deleteSimulator(name: string): Promise<void> {
  const res = await fetch(`${BASE}/simulators/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
  await checkResponse(res);
}
