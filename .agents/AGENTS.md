# Soul

You are a distinguised senior enginer who values simplicity, readability, reliability, and security. 

# Skills

When a task requires a skill:
1. Discover skills in `./skills/` (relative to this AGENTS.md file)
2. Read the target `SKILL.md` from `./skills/<skill>/SKILL.md`.
3. Load only the referenced files needed for the active task.

Do not create duplicate copies of skills under tool-specific folders.

# Miru

**What Miru is:** Miru is a configuration management solution for robotics teams. It provides tools for engineers to define, deploy, and manage application configurations for their robots at scale.

**What Miru aims to solve:** As robot fleets grow, configuration complexity grows with it—hardware differences, customer-specific tuning, and feature flags lead to many subtly different configs. Managing them is spread across the team (software engineers, support, field engineers) with different tools and skill levels. Miru addresses this by:

- **Tedious, error-prone workflows** — Replacing manual SSH and file edits with an approachable GUI and programmatic APIs so configs are managed and versioned in one place.
- **Difficult versioning** — Giving each device clear config state per release, with device-specific overrides and a known history (what ran when, what changed fleet-wide vs locally).
- **Lack of auditability** — Recording every change with author, timestamp, and diff so teams can trace how a robot arrived at its current state.
- **Ad-hoc overrides** — Providing structured config schemas, validation before deployment, and scoped overrides so the fleet has a defined source of truth instead of copied, diverging files.

Miru prioritizes reliability, security, and developer experience. All configuration management is accessible via APIs, Git integration, and SDKs for CI/CD and internal tools.

---

**Where to look for answers about Miru:** The easiest place to consult is the **MCP server** (Mintlify Documentation Search): it serves the current docs in production (main branch of the `docs` repo) and is the best first stop for product behavior, concepts, and API usage.

When **developing new features or changing product behavior**, you may need to consult or edit the **`/docs` repo** itself—for example to add or update guides, concepts, or reference docs so the public documentation stays in sync with the product.

## Git Repository Architecture

This document describes the git repository architecture of **Miru**: the repositories, how they relate, and how the system runs. 

## Repositories

All repos live under the `mirurobotics` GitHub org. The source of truth for repo configuration is `infra/github/terraform/repos.tf`.

### Product — deployed services and binaries

| Repo | Language | What it is |
|------|----------|------------|
| **backend** | Go | Main API server — auth, business logic, persistence. Deployed to EKS as a container. |
| **frontend** | TypeScript (Next.js) | User-facing dashboard — releases, devices, config, admin. Deployed on Vercel. |
| **agent** | Rust | Binary that runs on customer devices — applies config, reports state, talks to backend over HTTPS and MQTT. |
| **cli-private** | Go | `miru` CLI implementation — release, deploy, login, etc. Distributed as a binary. |

### Libraries and specs

| Repo | What it is |
|------|------------|
| **core** | Shared Go library — errors, logging, context, filesystem helpers, JSON/YAML, option types. Consumed by backend and cli-private. |
| **openapi** | OpenAPI specifications and codegen. Single source for API contracts consumed by backend, frontend, agent, and cli-private. |
| **analytics** | OpenAPI spec for analytics APIs. |
| **gotools** | Shared Go development tools (linters, generators) for Miru Go repositories. |

### Distribution and installation

| Repo | What it is |
|------|------------|
| **cli** | Public CLI distribution — installation scripts (e.g. `curl \| sh`). |
| **setup-cli** | GitHub Action for installing the Miru CLI in CI. |
| **homebrew-cli** | Homebrew tap for the Miru CLI. |
| **cli-sdk** | Stainless-generated SDK for CLI-backend client. |

### Documentation and marketing

| Repo | What it is |
|------|------------|
| **docs** | Mintlify documentation site — product docs, API references, guides. |
| **website** | Marketing/landing site (Lovable). |
| **getting-started** | Getting started guides and examples. |

### Infrastructure and tooling

| Repo | What it is |
|------|------------|
| **infra** | Terraform IaC — AWS (EKS, ECR, VPC, ACM), Cloudflare DNS, EMQX MQTT, GitHub org/repo config. |
| **tests** | End-to-end and integration test suites. |
| **workbench** | Meta repository for local development — ties repos together, houses shared AI context, plans, and research. |
| **ai** | Shared AI-assisted development context (rules, skills). Consumed by all repos via git subtree at `.agents/`. |
| **.github** | Org-wide community health and config files. |
| **metrics-dashboard** | Internal engineering and product metrics dashboard. |

## Runtime topology

```
                  +-----------+
                  |  backend  |  (EKS, Go)
                  +-----------+
                   ^   ^    ^
                   |   |    |
   +-----------+   |   |    |   +-----------+
   | frontend  |---+   |    +---|   agent   |
   +-----------+       |        +-----------+
    (Vercel)           |         (on device)
        ^              |              ^
        |              |              |  MQTT (EMQX Cloud)
   +-----------+       |         +-----------+
   |  Browser  |       +--------|    CLI     |
   +-----------+                +-----------+
  (user machine)               (user machine)
```

- **Users** interact via the **frontend** (dashboard) in a browser or the **CLI** on their machine. Both call the **backend** API.
- **Devices** run the **agent** binary, which talks to **backend** over HTTPS (reporting state, fetching config) and receives real-time updates via **MQTT** (EMQX Cloud).
- **docs** and **website** are deployed as static sites with no runtime backend dependency.

## Infrastructure

Managed via Terraform in the `infra` repo across three directories:

| Directory | Scope |
|-----------|-------|
| `deploy/terraform/` | AWS infrastructure — EKS cluster, VPC, ECR, ACM certificates, Cloudflare DNS (API and MQTT domains), ALB ingress. |
| `deploy/k8s/` | Kubernetes manifests per environment (production, staging, uat). Backend runs as a Deployment with a Service + Ingress behind an ALB. |
| `github/terraform/` | GitHub org config — repos, branch protection, collaborators, Dependabot. |
| `manage/terraform/` | GitHub Actions OIDC roles for CI/CD. |

### Key services

- **AWS EKS** — Kubernetes cluster (`miru-backend`) running the backend API server.
- **AWS ECR** — Container registry for backend images.
- **Cloudflare** — DNS and proxy for `mirurobotics.com` and `miruml.com` domains.
- **EMQX Cloud** — Managed MQTT broker for device communication (`mqtt.mirurobotics.com`).
- **Vercel** — Hosts frontend and website.
- **Mintlify** — Hosts docs.
- **Datadog** — Observability (logs, metrics) for the backend cluster.

### Environments

Three environments: **production**, **staging**, **uat**. Each has its own K8s namespace, DNS records, and EMQX instance. Backend is deployed via environment-specific K8s manifests.

## Dependency graph

```
openapi ──► backend ◄── core
   │            ▲
   ├──► frontend│
   │            │
   ├──► agent   │
   │            │
   └──► cli-private ◄── core
            │
            └──► cli-sdk
```

- **core** and **openapi** are foundations — they have no Miru product dependencies.
- **backend** depends on core; its API shape is driven by openapi.
- **frontend**, **agent**, and **cli-private** all consume openapi specs and call the backend API.
- **cli-private** also depends on core and cli-sdk.

## Workbench repository

The meta repo (`workbench`) exists to:

- Provide a unified workspace for development across Miru repos
- House shared AI context (`.agents/` subtree from `mirurobotics/ai`)
- Store cross-repo plans (`plans/`) and research (`research/`)
- Maintain workspace-wide tooling and scripts (`scripts/`)