# Frontend Design Patterns (Angular)

## 1. Module Architecture

The frontend follows a domain-based structure at `src/app/`, mirroring the backend modules. This keeps the mental model consistent across the stack and ensures zero cross-imports between domains.

```
src/app/
  shared/
    components/     # Reusable UI components (buttons, modals, terminal emulator)
    directives/     # Shared directives
    pipes/          # Pure pipes
    models/         # Cross-domain types, API response wrappers
    utils/          # Helper functions, constants
  user/
    models/         # User interfaces, types
    store/          # Signal-based state
    services/       # API calls + business logic
    pages/          # Route-level components
    components/     # Domain-specific components
  auth/
    models/
    store/
    services/
    pages/
    guards/         # Auth guards (canActivate, canMatch)
  agent/
    models/
    store/
    services/
    pages/
    components/
  session/
    models/
    store/
    services/
    pages/
    components/
  provider/
    models/
    store/
    services/
    pages/
    components/
```

The domains listed are the initial set. New domains can be added as the system grows — existing modules are never modified when adding a new one.

## 2. Domain Structure

Each sub-directory within a domain has a single responsibility:

| Directory | Owns |
|---|---|
| `models/` | TypeScript interfaces, type aliases, enums |
| `store/` | Hand-rolled signal stores — plain service classes with `signal()` and `computed()` fields |
| `services/` | API calls (`HttpClient`-based), optionally facade services combining multiple API calls with store updates |
| `pages/` | Route-level components (one per route) |
| `components/` | Domain-specific UI components consumed by `pages/` or other `components/` |

Shared or cross-domain pieces live in `shared/`. No domain imports directly from another domain's internals — if sharing is needed, the shared piece moves to `shared/`.

## 3. State Management with Signals

The project uses hand-rolled signal stores exclusively. No NgRx, no `BehaviorSubject`, no legacy RxJS-based state management.

### Store pattern

```typescript
@Injectable({ providedIn: 'root' })
export class AgentStore {
  readonly agents = signal<Agent[]>([]);
  readonly selectedAgent = signal<Agent | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  readonly isEmpty = computed(() => this.agents().length === 0);
  readonly hasError = computed(() => this.error() !== null);

  constructor(private service: AgentService) {}

  async refresh(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);
    try {
      const agents = await firstValueFrom(this.service.getAgents());
      this.agents.set(agents);
    } catch (e) {
      this.error.set('Failed to load agents');
    } finally {
      this.loading.set(false);
    }
  }
}
```

### Rules

- Stores are `providedIn: 'root'` singletons
- No client-side mutation of server state — POST/PUT/DELETE calls the API, then calls `refresh()`
- `effect()` is used sparingly — logging, localStorage sync, timers — never for data flow
- `computed()` for derived state only, not for triggering side effects

## 4. Data Fetching with `resource()`

The Angular `resource()` utility is the primary mechanism for data fetching. It provides a reactive, signal-native contract without introducing external dependencies.

**The API is the source of truth, not the client.** There is no client-side cache. Data is fetched fresh on every meaningful trigger (navigation, user action, periodic poll).

### Basic pattern

```typescript
// agent/agents-page.component.ts
@Component({ ... })
export class AgentsPageComponent {
  readonly store = inject(AgentStore);

  private readonly refreshTrigger = signal(0);

  readonly agentsResource = resource({
    request: () => ({ refresh: this.refreshTrigger() }),
    loader: async ({ request }) => {
      this.store.loading.set(true);
      try {
        const agents = await firstValueFrom(this.store.service.getAgents());
        this.store.agents.set(agents);
        return agents;
      } finally {
        this.store.loading.set(false);
      }
    },
  });

  refresh(): void {
    this.refreshTrigger.update(v => v + 1);
  }
}
```

### Loading strategies

- **On navigation**: route resolver or `ngOnInit` calls `store.refresh()` or triggers the resource
- **Periodic polling**: a simple timer in the store or component calls `refresh()` on an interval
- **On user action**: the action handler calls the API, then calls `store.refresh()`

### When to use TanStack Query

`resource()` covers the common cases. If the project later needs cache deduplication, automatic background refetch, or optimistic mutations, `@tanstack/angular-query` can be introduced for specific domains without changing the overall architecture. The store pattern remains the same — only the fetch mechanism swaps.

## 5. Component Patterns

### Standalone components

All components are standalone. No `NgModule` wrappers.

```typescript
@Component({
  selector: 'app-agent-card',
  standalone: true,
  imports: [ ... ],
  templateUrl: './agent-card.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AgentCardComponent {
  readonly agent = input.required<Agent>();
  readonly selected = output<Agent>();
}
```

### Rules

- `input()` / `output()` signals only — no `@Input()` or `@Output()` decorators
- `OnPush` change detection on every component
- No manual `ChangeDetectorRef` — signals handle propagation
- Templates use the `@let` syntax for intermediate computations

## 6. Service Patterns

### API service

A thin, typed wrapper over `HttpClient`:

```typescript
@Injectable({ providedIn: 'root' })
export class AgentService {
  private readonly http = inject(HttpClient);

  getAgents(): Observable<Agent[]> {
    return this.http.get<Agent[]>('/api/agents');
  }

  getAgent(id: number): Observable<Agent> {
    return this.http.get<Agent>(`/api/agents/${id}`);
  }

  createAgent(data: CreateAgentRequest): Observable<Agent> {
    return this.http.post<Agent>('/api/agents', data);
  }

  deleteAgent(id: number): Observable<void> {
    return this.http.delete<void>(`/api/agents/${id}`);
  }
}
```

### Facade service

Optional, used when a single interaction spans multiple API calls and store updates:

```typescript
@Injectable({ providedIn: 'root' })
export class AgentFacade {
  private readonly store = inject(AgentStore);
  private readonly service = inject(AgentService);

  async createAndRefresh(data: CreateAgentRequest): Promise<void> {
    await firstValueFrom(this.service.createAgent(data));
    await this.store.refresh();
  }
}
```

No business logic in services — that belongs in the Go backend.

## 7. Routing

### Lazy-loaded per-domain routes

```typescript
// app.routes.ts
export const routes: Routes = [
  { path: '', redirectTo: '/agents', pathMatch: 'full' },
  { path: 'login', loadChildren: () => import('./auth/auth.routes') },
  { path: 'agents', loadChildren: () => import('./agent/agent.routes'), canMatch: [AuthGuard] },
  { path: 'admin', loadChildren: () => import('./user/user.routes'), canMatch: [AdminGuard] },
];

// agent/agent.routes.ts
export default [
  { path: '', component: AgentListPageComponent },
  { path: ':id', component: AgentDetailPageComponent },
] as Routes;
```

### Navigation and data loading

- Route resolvers call `store.refresh()` and return the promise to block navigation until data is loaded
- Alternatively, the page component calls `refresh()` in its constructor or `ngOnInit` and shows a loading state — preferred when incremental loading is acceptable

## 8. Testing

| Layer | Approach |
|---|---|
| **Services** | `HttpTestingController` — verify request method, URL, headers; provide mock responses |
| **Stores** | Unit-testable — instantiate directly, call methods, assert `signal()` values |
| **Components** | Provide mock store signals, verify rendered output with `fixture.nativeElement` |
| **Pages** | Use `provideHttpClient(withInterceptors(...))` testbed for route-level integration |

### Store test example

```typescript
describe('AgentStore', () => {
  let store: AgentStore;
  let service: jasmine.SpyObj<AgentService>;

  beforeEach(() => {
    service = jasmine.createSpyObj('AgentService', ['getAgents']);
    store = new AgentStore(service);
  });

  it('loads agents into signal', async () => {
    const mockAgents = [{ id: 1, name: 'Test' } as Agent];
    service.getAgents.and.returnValue(of(mockAgents));
    await store.refresh();
    expect(store.agents()).toEqual(mockAgents);
    expect(store.loading()).toBe(false);
  });
});
```

## 9. Scalability & Replaceability

| Concern | How the architecture addresses it |
|---|---|
| **Domain isolation** | Zero cross-imports between domains — a `user/` change never touches `agent/` |
| **Backend swap** | Store talks to service interface; swapping the backend is replacing the service implementation |
| **State library swap** | Hand-rolled signals have no runtime dependency — business logic is just TypeScript |
| **Bundle size** | No NgRx, no Redux, no external state library |
| **Team scaling** | Each domain is a self-contained directory; ownership boundaries are physical |
| **Feature growth** | New domain = new folder at `src/app/`, new route config, zero changes to existing code |

## 10. Styling with TailwindCSS

### 10.1 Approach

- All styling uses Tailwind utility classes — no component-scoped CSS files or CSS-in-JS
- Tailwind v4 CSS-first configuration via `@theme` in `src/styles.css`
- Dark mode via the `dark` variant, triggered by the `dark` class on `<html>`

### 10.2 Theme Service

A simple service manages the dark mode class and syncs with the system preference:

```typescript
// shared/utils/theme.service.ts
import { BreakpointObserver } from '@angular/cdk/layout';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  readonly isDark = signal(false);

  constructor(private breakpointObserver: BreakpointObserver) {
    const saved = localStorage.getItem('theme');
    if (saved) {
      this.setTheme(saved === 'dark');
    } else {
      this.breakpointObserver
        .observe('(prefers-color-scheme: dark)')
        .subscribe((state) => this.setTheme(state.matches));
    }
  }

  toggle(): void {
    const next = !this.isDark();
    localStorage.setItem('theme', next ? 'dark' : 'light');
    this.setTheme(next);
  }

  private setTheme(dark: boolean): void {
    document.documentElement.classList.toggle('dark', dark);
    this.isDark.set(dark);
  }
}
```

The service is injected once at app bootstrap. It restores a saved preference, or defaults to the system `prefers-color-scheme` and listens for live changes.

### 10.3 Semantic Color Tokens

Instead of hardcoding colors per component, semantic tokens are defined once in the CSS and referenced by all templates. This ensures color consistency across the entire application in both light and dark mode without developers thinking about dark/light variants in each template.

```css
/* src/styles.css */
@import "tailwindcss";

@theme {
  --color-surface-base: #ffffff;
  --color-surface-raised: #f9fafb;
  --color-surface-overlay: #ffffff;
  --color-surface-hover: #f3f4f6;

  --color-text-primary: #111827;
  --color-text-secondary: #4b5563;
  --color-text-muted: #9ca3af;
  --color-text-inverse: #ffffff;

  --color-border-default: #e5e7eb;
  --color-border-strong: #d1d5db;
  --color-border-focus: #6366f1;

  --color-accent-primary: #6366f1;
  --color-accent-hover: #4f46e5;
  --color-accent-muted: #eef2ff;

  --color-status-success: #22c55e;
  --color-status-warning: #f59e0b;
  --color-status-error: #ef4444;
  --color-status-info: #3b82f6;
}
```

### 10.4 Dark Mode Overrides

```css
@variant dark {
  --color-surface-base: #0f172a;
  --color-surface-raised: #1e293b;
  --color-surface-overlay: #1e293b;
  --color-surface-hover: #334155;

  --color-text-primary: #f1f5f9;
  --color-text-secondary: #94a3b8;
  --color-text-muted: #64748b;
  --color-text-inverse: #0f172a;

  --color-border-default: #334155;
  --color-border-strong: #475569;
  --color-border-focus: #818cf8;

  --color-accent-primary: #818cf8;
  --color-accent-hover: #6366f1;
  --color-accent-muted: #1e1b4b;
}
```

Status colors are omitted intentionally — they stay consistent across both modes since their semantics (error is red, success is green) do not change with the theme.

### 10.5 Usage Convention

Templates reference these semantic tokens directly. No conditional `dark:` variants are needed at the usage site — the tokens auto-switch based on the `dark` class on `<html>`:

```html
<div class="bg-surface-base border border-border-default rounded-lg p-4">
  <h2 class="text-text-primary font-semibold">Agent Name</h2>
  <p class="text-text-secondary text-sm">Status: running</p>
</div>

<button class="bg-accent-primary text-text-inverse px-4 py-2 rounded hover:bg-accent-hover">
  Create Agent
</button>

<span class="text-status-success">● Running</span>
<span class="text-status-error">● Error</span>
```

### 10.6 When to Add New Tokens

Only add a new token when a distinct semantic role appears in the design. For one-off needs, compose an existing token. Over-tokenization adds configuration surface without improving consistency.
