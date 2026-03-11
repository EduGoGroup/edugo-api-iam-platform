# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.14.0] - 2026-03-11

### Tipo de Release: patch

- chore: update dependencies in go.mod and go.sum to latest versions
- feat: UI review improvements — screen config, menu service, sync service updates
- update (#27)
- chore: bump edugo-infrastructure/postgres to v0.61.0 (#26)
- Use pgx SimpleProtocol to fix prepared statement issues
- feat: glossary bucket in sync bundle (#24)
- fix: auth service and debug config improvements (#22)
- feat(iam-platform): authorization guards + audit trail + login tracking (#20)

---

## [0.13.0] - 2026-03-07

### Tipo de Release: patch

- update (#25)

---

## [0.12.0] - 2026-03-06

### Tipo de Release: patch



---

## [0.11.0] - 2026-03-06

### Tipo de Release: patch

- update (#23)
- feat(iam-platform): authorization guards + audit trail + login tracking (#20) (#21)
- skill
- Add pagination, is_active filter, and 400 responses to docs

---

## [0.10.0] - 2026-03-04

### Tipo de Release: patch

- chore: bump repository to v0.3.3 (ApplyPagination + int64 totals)
- chore(deps): bump shared/common to v0.52.0
- fix(pagination): address code review comments from PR #18
- feat(pagination): implement real pagination with COUNT for IAM endpoints
- perf(docker): eliminate Go compilation from Docker, reduce image time ~80%

---

## [0.9.0] - 2026-03-03

### Tipo de Release: patch

- fix(tests): remove ResourceKey from Permission struct literals
- chore(deps): bump edugo-shared/auth to v0.52.0 and repository to v0.3.2
- fix: address PR review comments on permissions and refresh token
- chore(deps): bump edugo-infrastructure/postgres to v0.58.0
- feat(permissions): fix CRUD, status filter chips, refresh token school context

---

## [0.8.0] - 2026-03-02

### Tipo de Release: patch

- fix: correct json.RawMessage fields typed as object in Swagger docs
- Add CRUD endpoints for permissions to API docs and update deps

---

## [0.7.0] - 2026-03-02

### Tipo de Release: patch

- chore: upgrade edugo-infrastructure/postgres to v0.54.0
- fix: address PR review comments - validation, tests, bulk insert
- feat: CRUD roles, permissions and role_permissions management (Fase 1.1-1.3)

---

## [0.6.0] - 2026-02-27

### Tipo de Release: patch

- Sort contexts and user roles by school, role, and academic unit

---

## [0.5.0] - 2026-02-27

### Tipo de Release: patch

- feat: add buckets filter to sync/bundle endpoint for progressive loading

---

## [0.4.0] - 2026-02-26

### Tipo de Release: patch

- chore: upgrade edugo-shared/auth to v0.51.0
- fix: correct swagger response type for GetScreenVersion endpoint
- fix: address code review — error handling, deterministic responses, swagger
- feat: Sprint 8 — sync bundle, deterministic hashes, screen versioning

---

## [0.3.0] - 2026-02-26

### Tipo de Release: patch

- fix: address code review issues from PR#5 search/filter feature
- Add search and filtering support to list endpoints and update dependencies

---

## [0.2.0] - 2026-02-25

### Tipo de Release: patch

- Remove unused fields from CreateInstanceRequest and docs

---

## [0.1.2] - 2026-02-24

### Tipo de Release: patch

- fix: allow super_admin to switch context without school membership
- fix: use GITHUB_TOKEN instead of GHCR_TOKEN for registry auth

---

## [0.1.1] - 2026-02-24

### Tipo de Release: patch



---

## [0.1.0] - 2026-02-24

### Tipo de Release: patch

- Remove config directory copy from Dockerfile
- Update edugo-shared/repository to use v0.1.0 module
- chore: release v0.1.0
- Use secrets for Postgres config in Azure deploy workflow
- Update dependencies and remove unused hashToken function
- docs: add CHANGELOG and version tracking
- feat: add GitHub Actions CI/CD workflows
- feat: add Docker and Make build configuration
- feat: initial commit - IAM Platform API

---

## [0.1.0] - 2026-02-24

### Tipo de Release: patch

- Use secrets for Postgres config in Azure deploy workflow
- Update dependencies and remove unused hashToken function
- docs: add CHANGELOG and version tracking
- feat: add GitHub Actions CI/CD workflows
- feat: add Docker and Make build configuration
- feat: initial commit - IAM Platform API

---

---

## [Unreleased]

### Added
- Initial project structure
- Authentication and authorization modules
- User, role, and permission management
- Screen configuration and menu management
- PostgreSQL persistence layer
- Swagger documentation
- CORS middleware and error handling
- Docker support with multi-stage builds
- Makefile for development and CI/CD
- GitHub Actions workflows for testing and deployment
