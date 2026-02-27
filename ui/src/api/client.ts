export interface Simulator {
  name: string;
  status?: string; // reserved for future backend extension
}

export interface CreateSimulatorRequest {
  name: string;
  dump_payload: string;   // matches Go json tag
  rules_payload: string;  // matches Go json tag
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
  const names: string[] = await res.json();
  return names.map((name) => ({ name }));
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
