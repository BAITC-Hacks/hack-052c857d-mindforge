# Аким на 5 часов

An AI-powered city management simulator, planned as a small hackathon MVP. This repository currently contains project scaffolding only; application behavior has not been implemented.

## Planned scope

The simulator is planned to model five development categories: transport, green infrastructure, social infrastructure, safety, and city services. Future simulation inputs may include a fixed virtual budget, districts and their indicators, and initiatives with costs and effects. The planned deterministic simulation will produce an Astana Quality of Life Score and results for later AI analysis.

## Project layout

- `backend/` — Go API entry point and backend code.
- `ai/` — Python package for future analysis of simulation results.
- `data/synthetic/` — synthetic city datasets used during development.
- `tests/` — backend and AI test locations.
- `config/` — local, non-secret project configuration templates.

The scaffold intentionally has no web framework, database, container setup, or external service dependencies.

## Development

- Go module: `backend/go.mod`
- Python module: `ai/pyproject.toml`

