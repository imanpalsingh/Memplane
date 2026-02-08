# Memplane

Memplane is a production-oriented, standalone episodic memory service for LLM systems.

It is inspired by the paper:
- [HUMAN-INSPIRED EPISODIC MEMORY FOR INFINITE CONTEXT LLMS](https://arxiv.org/pdf/2407.09450)

## Why Memplane

Long-context applications need memory that is:
- event-structured, not just raw chunk storage,
- retrieval-aware for relevance and temporal continuity,
- deployable as an independent service for framework integrations.

Memplane is being built to serve that role for ecosystems such as LangChain and Mastra.

## Project Status

Current phase: bootstrap and service foundation.

## Quick Start

```bash
go test ./...
go vet ./...
go run ./cmd/memplane
```

Health check:

```bash
curl -i http://127.0.0.1:8080/health
```

## Roadmap

1. Service foundation (done)
2. Core episodic event model
3. In-memory event store
4. Surprise-based boundary detection (paper-aligned)
5. Boundary refinement (paper-aligned)
6. Two-stage retrieval: similarity + contiguity buffers
7. Public API for ingest/retrieve/session lifecycle
8. Durable persistence
9. LangChain and Mastra adapters
10. Evaluation harness and production hardening
