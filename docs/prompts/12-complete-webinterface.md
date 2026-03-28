Let's build Phase 3: The Web Interface for MetricForge. We need a modern, single-page React application using Vite, TypeScript, and Tailwind CSS. 

Requirements:

1. **Setup & API Client (`/ui/src/api/client.ts`)**:
   - Initialize a Vite React+TS project if not already done, and configure Tailwind CSS.
   - Define TypeScript interfaces:
     - `Mutator { type: string; params: Record<string, any> }`
     - `RuleMatch { metric_name: string; labels?: Record<string, string> }`
     - `Rule { name: string; match: RuleMatch; mutator: Mutator }`
     - `Simulator { name: string; status: string; active_rules: Rule[] }`
   - Write `fetch` wrapper functions: `getSimulators()`, `createSimulator(req)`, and `deleteSimulator(name)`.

2. **Create `CreateForm.tsx`**:
   - A form with: `Name` (input), `Metrics Dump` (textarea), and `Mutation Rules` (textarea).
   - Use React state. Include a "Deploy Simulator" button that disables while `loading` is true.
   - Accept an `onSubmit` prop to pass data up.

3. **Create `SimulatorList.tsx`**:
   - Accept a `simulators` array prop and an `onDelete(name)` function.
   - Render a Tailwind-styled card for each simulator. The header shows the name, status, and a red "Delete" button.
   - **Expandable Details:** Make the card expandable. When expanded, show a clean table or nested list of the `active_rules`.
   - **Rule Display:** For each rule, display:
     - Column 1: The Rule Name and `metric_name` (with any labels formatted nicely).
     - Column 2: The Mutator Type using color-coded Tailwind badges (e.g., purple for 'spike', blue for 'wave', orange for 'jitter').
     - Column 3: The Parameters. Iterate over the `params` object and display them cleanly (e.g., `variance: 0.05`, `multiplier: 10, duration: 30s`).
   - Show a "No simulators running" empty state if the array is empty.

4. **Update `App.tsx`**:
   - Create a two-column layout: Form on the left, List on the right. Top nav bar says "MetricForge Control Plane".
   - Use `useEffect` to fetch simulators on mount. Manage `simulators`, `isLoading`, and `error` state.
   - Wire up the submit and delete handlers to refresh the list automatically. Display API errors in a styled alert at the top.
