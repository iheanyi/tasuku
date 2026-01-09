---
name: decide
description: Record an architectural decision with reasoning. Use when making design choices, selecting technologies, or resolving trade-offs.
---

# Record Decision

Capture important decisions with the alternatives considered and reasoning.

## Usage

```bash
tk decision add --id auth-strategy --chose "JWT tokens" --over "sessions,OAuth" --because "Stateless for microservices"
tk decision list                    # List all decisions
tk decision remove auth-strategy    # Remove a decision
```

## What to Record

Good decisions to document:

- **Technology choices**: "Chose PostgreSQL over MongoDB for relational data"
- **Architecture patterns**: "Chose event sourcing over CRUD for audit trail"
- **API design**: "Chose REST over GraphQL for simplicity"
- **Trade-offs**: "Chose performance over memory efficiency"

## Decision Structure

Each decision includes:
- **ID**: Identifier for the decision
- **Chose**: What was selected
- **Over**: Alternatives that were considered
- **Because**: Reasoning behind the choice

## When to Use

- Selecting a framework or library
- Designing system architecture
- Choosing between implementation approaches
- Making trade-offs that affect future work

## Best Practices

1. Record decisions before implementing them
2. Include realistic alternatives that were considered
3. Be specific about the reasoning
4. Reference decisions in related tasks
