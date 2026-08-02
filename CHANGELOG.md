# Changelog

All notable changes to this project are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Admin export/import: file size shown on selection (import) and after generation (export), with a warning if it exceeds the configured limit.
- Admin → Paramètres: import/export size limit (`MAX_IMPORT_SIZE_MB`) is now adjustable live, no redeploy needed.

### Fixed
- Nginx's default 1MB body-size limit silently rejected export/import uploads before they ever reached the backend; now set well above the backend's cap. Import failures are now surfaced as an on-page error instead of failing silently.

## [0.2.0] - 2026-08-01

### Added
- Admin section: export/import of recipes, cocktails, event history and users (including password hash) as a portable JSON bundle, with a preview step before importing.
- Admin section: "Mise à jour" — versioned SQL migrations for schema changes AutoMigrate can't safely do on its own (renames, backfills), applied one at a time with a dry-run step. See `UPGRADE.md`.

### ⚠️ Base de données
- Adds a `SCHEMA_VERSION` setting (defaults to `1` on existing installs, no action needed).
- Demo migration `0002_add_demo_notes_column` (harmless, inert column) is available in Admin → Mise à jour to verify the pipeline — see `UPGRADE.md` Cas 2. Confirmed backward-compatible: the site stayed fully functional after applying it, before any code/`REPO_REF` change.

### Changed
- Fresh deployments no longer auto-seed demo recipes/cocktails (Mojito, Margarita, etc.). Existing deployments keep whatever was already seeded.

### Fixed
- Ingredient autocomplete now also works on non-voted (organizer) matrix lists, not just voted ones.

## [0.1.0] - 2026-08-01

First tagged release. CampMenu was already in daily use before this point;
this tag marks the start of versioned releases going forward.

### Added
- `campmenu-dev` staging namespace and `dev` branch, so changes can be
  validated in a real deployed environment before reaching `main`/prod.
- Ingredient autocomplete (fuzzy suggestions against the shared ingredient
  referential) on the desktop matrix "new article" input.
- New lists created from within an event default to "event-specific" scope.

### Fixed
- Ad-hoc/mobile shopping items were being saved as voted lists by default
  (GORM zero-value gotcha).
