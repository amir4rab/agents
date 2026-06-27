# AI Rules

## Core Principles

Every abstraction must solve an existing problem.
Prefer concrete implementations over indirection.

The codebase prioritizes:
- Simplicity
- Readability
- Maintainability
- Explicitness
- Testability

The codebase does not prioritize:
- Architectural purity
- Pattern Completeness
- Future hypothetical requirements
- Abstractions without need

## Global Rules

Always:

- Follow existing patterns
- Prefer consistency over innovation
- Prefer explicit code over magic
- Keep implementations cohesive
- Reuse existing code when appropriate

Never:

- Introduce new frameworks
- Introduce new architectural patterns without permission
- Refactor unrelated code
- Add complexity without justification

## Abstraction Rules

Do not create:

Interfaces are allowed when:

- Defining domain boundaries
- Supporting testing 
- Multiple implementation exist

Prefer Concrete implementations whenever possible.