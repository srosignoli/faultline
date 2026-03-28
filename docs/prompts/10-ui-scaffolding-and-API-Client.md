Let's start Phase 3: The Web Interface. We are going to build a React application using Vite, TypeScript, and Tailwind CSS to control PromSim.

First, I need you to scaffold the project and build the API client. 

Requirements:
1. Initialize a new Vite project in the `/ui` directory using the React + TypeScript template.
2. Install and configure Tailwind CSS for the `/ui` project according to the official Vite+Tailwind installation guide. Ensure the `tailwind.config.js` and `index.css` files are correctly set up.
3. Create a new directory `/ui/src/api` and define a TypeScript file `client.ts`.
4. In `client.ts`, define the TypeScript interfaces that match our Go backend:
   - `Simulator` (Name, Status, etc.)
   - `CreateSimulatorRequest` (Name, DumpPayload, RulesPayload)
5. Implement an `ApiClient` class or a set of async functions using the native browser `fetch` API to interact with our backend:
   - `getSimulators(): Promise<Simulator[]>` -> GET `/api/simulators`
   - `createSimulator(req: CreateSimulatorRequest): Promise<void>` -> POST `/api/simulators`
   - `deleteSimulator(name: string): Promise<void>` -> DELETE `/api/simulators/${name}`
6. Add error handling to these fetch calls so they throw meaningful errors if the backend returns a 400 or 500 status code.
7. Only output the scaffolding commands, configuration files, and the `client.ts` file. Do not build the React UI components yet.
