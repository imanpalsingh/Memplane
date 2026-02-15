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

Current phase: episodic event primitives, surprise-based segmentation with modularity refinement, and anchor-based retrieval API.

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

Create event:

```bash
curl -i -X POST http://127.0.0.1:8080/v1/events \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"evt_1","tenant_id":"tenant_1","session_id":"session_1","start_token":0,"end_token_exclusive":10,"created_at":"2026-02-10T12:00:00Z"}'
```

List session events:

```bash
curl -i "http://127.0.0.1:8080/v1/events?tenant_id=tenant_1&session_id=session_1"
```

Segment from surprise scores:

```bash
curl -i -X POST http://127.0.0.1:8080/v1/segment \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"tenant_1","session_id":"session_1","start_token":100,"surprise":[0.05,0.2,1.2,0.1,0.15,1.5,0.2],"key_similarity":[[1,0,0,0,0,0,0],[0,1,0,0,0,0,0],[0,0,1,0,0,0,0],[0,0,0,1,0,0,0],[0,0,0,0,1,0,0],[0,0,0,0,0,1,0],[0,0,0,0,0,0,1]],"threshold":0.8,"min_boundary_gap":1,"created_at":"2026-02-14T12:00:00Z","event_id_prefix":"seg"}'
```

Retrieve around anchor events:

```bash
curl -i -X POST http://127.0.0.1:8080/v1/retrieve \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"tenant_1","session_id":"session_1","event_ids":["seg_1"],"top_k":1,"buffer_before":1,"buffer_after":1}'
```

## Roadmap

1. Service foundation (done)
2. Core episodic event model (done)
3. In-memory event store (done)
4. Initial ingest/list API (done)
5. Surprise-based boundary detection (paper-aligned, done)
6. Boundary refinement (paper-aligned, done)
7. Two-stage retrieval: similarity + contiguity buffers (in progress)
8. Durable persistence
9. LangChain and Mastra adapters
10. Evaluation harness and production hardening
