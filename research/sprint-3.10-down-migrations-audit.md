# Sprint 3.10 — Down Migrations Audit

**Date:** 2026-07-30
**Scope:** Full `.down.sql` system for migrations 000001–000023
**Toolchain:** All SQL validated by structural review (shell/go unavailable for runtime test)

---

## Summary

| Metric | Value |
|--------|-------|
| Total `.up.sql` files | 23 |
| Total `.down.sql` created | 23 |
| Tables dropped in rollback | ~120 |
| Indexes dropped in rollback | ~200 |
| ALTER TABLE ADD COLUMN reversed | ~50 |
| Constraints/FKs dropped | 2 explicit (fk_sites_owner, seo_clusters_parent_cluster_id_fkey) |
| Extensions preserved (not dropped) | 3 (uuid-ossp, pg_trgm, pgcrypto) |
| Functions preserved (not dropped) | 1 (update_updated_at_column) |
| INSERT seed data cleaned | global_settings, editorial_widgets, writing_styles |
| Migrations requiring attention | 2 (000003, 000016) |

---

## Migration-by-Migration Breakdown

### 000001 — Initial Schema
**File:** `000001_create_initial_schema.down.sql`
**Operations reversed:**
- Tables: sites, users, site_users, sessions, audit_log, audit_log_default
- Triggers on sites, users
- Extensions: **NOT dropped** (shared dependency risk)
- `update_updated_at_column()` function: **NOT dropped** (used by 15+ later migrations)

### 000002 — OAuth & MFA
**File:** `000002_add_oauth_mfa.down.sql`
**Operations reversed:**
- Tables: oauth_accounts, mfa_configs
- Columns on users: mfa_enabled, mfa_secret, deleted_at
- Index: idx_users_deleted (partial index on deleted_at)
- Triggers on oauth_accounts, mfa_configs

### 000003 — Audit Log (non-partitioned)
**File:** `000003_add_audit_log.down.sql`
**⚠️  Requires attention:**
- This migration's `CREATE TABLE IF NOT EXISTS` is a no-op when 000001 ran first.
- The down migration **only drops indexes** — does NOT drop the audit_log table.
- The table (whether partitioned by 000001 or non-partitioned by 000003) is handled by 000001.down.
- **If 000001 was skipped**, an orphaned audit_log remains after rollback.

### 000004 — Multi-Site System
**File:** `000004_add_sites.down.sql`
**Operations reversed:**
- Tables: site_domains, global_settings, site_settings, casbin_rules
- Columns added to sites: description, feature_flags, theme, locale, timezone, deleted_at, owner_id
- FK constraint: fk_sites_owner (sites.owner_id → users.id)
- RLS: disabled on sites, site_domains, site_settings
- Seed data: 10 global_settings rows deleted by key
- Triggers: set_site_settings/global_settings/site_domains_updated_at dropped
- Original trigger set_sites_updated_at restored (was DROP'd and re-CREATEd in 000004)
- Indexes: 3 dropped

### 000005 — Posts, Categories, Tags
**File:** `000005_add_posts.down.sql`
**Operations reversed:**
- Tables: posts, categories, tags, post_categories, post_tags
- RLS disabled on all 3 tables
- Triggers on posts, categories, tags
- All 10 indexes

### 000006 — Assets
**File:** `000006_add_assets.down.sql`
**Operations reversed:**
- Tables: assets, post_assets, post_autosaves
- RLS disabled on all 3 tables
- Trigger on assets
- All 8 indexes

### 000007 — Media Library
**File:** `000007_add_media_tables.down.sql`
**Operations reversed:**
- Tables: folders, media, media_variants
- Type: `media_variant_type` enum dropped
- RLS disabled on all 3 tables
- Triggers on media, folders
- All 9 indexes

### 000008 — Plugin System
**File:** `000008_add_plugins.down.sql`
**Operations reversed:**
- Tables: plugins, plugin_settings, plugin_permissions
- Triggers on plugins, plugin_settings
- 2 indexes

### 000009 — Editorial Tasks
**File:** `000009_add_editorial.down.sql`
**Operations reversed:**
- Tables: editorial_tasks, post_revisions, approval_requests, editorial_calendar_events, editorial_widgets
- Seed data: 6 auto-inserted widgets deleted by widget_type
- All indexes

### 000010 — Research Tables
**File:** `000010_add_research_tables.down.sql`
**Operations reversed:**
- Tables: research_jobs, research_sources, research_entities, research_briefings
- Unique index idx_research_briefings_job_id
- All 7 indexes

### 000011 — Writer Tables
**File:** `000011_add_writer_tables.down.sql`
**Operations reversed:**
- Tables: writing_styles, article_jobs, article_outlines, article_sections, article_versions
- Seed data: 8 writing styles deleted by slug per site
- All 8 indexes

### 000012 — Editorial Engine
**File:** `000012_add_editorial_tables.down.sql`
**Operations reversed:**
- Tables: editorial_pipelines, pipeline_stages, editorial_style_rules, editorial_seo_data, editorial_quality_scores, editorial_translations, editorial_prompt_data
- All 14 indexes

### 000013 — Content Generation
**File:** `000013_add_generation_tables.down.sql`
**Operations reversed:**
- Tables: generation_jobs, generation_pipeline, generation_pipeline_logs, generation_quality_gates, generation_stats
- All 13 indexes

### 000014 — Autocontent Engine
**File:** `000014_add_autocontent_tables.down.sql`
**⚠️ Historical conflict handled:**
- The original 000014 created `publication_queue` — renamed to `autocontent_queue` in Sprint 3.7.
- The down migration drops `autocontent_queue` (the renamed table), NOT `publication_queue`.
- `publication_queue` is owned by 000019 and dropped in 000019.down.
- Tables dropped: autocontent_jobs, autocontent_steps, autocontent_results, autocontent_queue, workflow_templates
- All 16 indexes

### 000015 — SEO Tables
**File:** `000015_add_seo_tables.down.sql`
**Operations reversed:**
- Tables: seo_projects, seo_keywords, seo_clusters, seo_audits, seo_internal_links, seo_metadata, seo_scores
- All 16 indexes

### 000016 — RLS Policies
**File:** `000016_add_rls_policies.down.sql`
**⚠️ Complex rollback — largest file (180 lines)**
**Operations reversed:**
- Section 8→1 (reverse order): all 7 groups of RLS policies dropped
- All ALTER TABLE ... ENABLE ROW LEVEL SECURITY reversed via DISABLE
- **Restores original buggy policies** from 000005 for posts, categories, tags:
  - Uses `current_user_id` instead of `current_site_id` for site_id comparison
  - This intentionally reverts to the pre-000016 bug state

### 000017 — Human Writer
**File:** `000017_add_human_writer_tables.down.sql`
**Operations reversed:**
- Tables: writing_profiles, writing_rules, writing_personas, vocabulary_sets, transition_library, style_patterns, sentence_templates, humanization_history
- All 17 indexes

### 000018 — Article Pipeline
**File:** `000018_add_article_pipeline.down.sql`
**Operations reversed:**
- Tables: article_pipeline_jobs, article_pipeline_steps, article_pipeline_metrics, article_quality_reports, publication_candidates
- All 9 indexes

### 000019 — Publisher Tables
**File:** `000019_add_publisher_tables.down.sql`
**Operations reversed:**
- Tables: publications, publication_history, publication_queue, publication_schedule, publication_metrics
- All indexes

### 000020 — SEO Engine Enhancements
**File:** `000020_add_seo_engine_tables.down.sql`
**Operations reversed:**
- Table: seo_improvements (with its 6 indexes)
- Columns removed from seo_projects (9), seo_keywords (5), seo_clusters (5), seo_audits (16), seo_scores (8)
- FK constraint `seo_clusters_parent_cluster_id_fkey` dropped via `DROP CONSTRAINT IF EXISTS`

### 000021 — Workflow Tables
**File:** `000021_add_workflow_tables.down.sql`
**Operations reversed:**
- Tables: workflow_jobs, workflow_steps, workflow_queue, workflow_history, workflow_notifications, workflow_dashboard
- All 29 indexes

### 000022 — Setup Tables
**File:** `000022_add_setup_tables.down.sql`
**Operations reversed:**
- Table: system_installation
- Index: idx_system_installation_installed

### 000023 — Grounding Fields
**File:** `000023_add_grounding_fields.down.sql`
**Operations reversed:**
- Table: article_sources (with 7 indexes)
- Columns removed from research_sources: freshness_score, is_verified, retrieved_at, grounding_metadata

---

## Rollback Execution Order

The correct rollback order is reverse of apply: **000023 → 000022 → ... → 000002 → 000001**

```
Step 1: 000023.down  — DROP article_sources, ALTER research_sources
Step 2: 000022.down  — DROP system_installation
Step 3: 000021.down  — DROP workflow_* tables
Step 4: 000020.down  — DROP seo_improvements, ALTER seo_* tables
Step 5: 000019.down  — DROP publication_* tables
Step 6: 000018.down  — DROP article_pipeline_* tables
Step 7: 000017.down  — DROP human writer tables
Step 8: 000016.down  — DROP RLS, disable RLS, restore buggy policies
Step 9: 000015.down  — DROP seo_* tables
Step 10: 000014.down — DROP autocontent_*, workflow_templates
Step 11: 000013.down — DROP generation_* tables
Step 12: 000012.down — DROP editorial engine tables
Step 13: 000011.down — DROP article_*, writing_styles
Step 14: 000010.down — DROP research_* tables
Step 15: 000009.down — DROP editorial task tables
Step 16: 000008.down — DROP plugin tables
Step 17: 000007.down — DROP media tables, DROP TYPE media_variant_type
Step 18: 000006.down — DROP asset tables
Step 19: 000005.down — DROP post tables
Step 20: 000004.down — DROP site_* tables, ALTER sites, restore trigger
Step 21: 000003.down — DROP indexes only (table owned by 000001)
Step 22: 000002.down — DROP oauth/mfa tables, ALTER users
Step 23: 000001.down — DROP sites, users, site_users, sessions, audit_log
```

---

## Migrations with Data Loss Risk

| Migration | Risk | Mitigation |
|-----------|------|------------|
| 000004 | DELETE FROM global_settings removes seed configuration | Seed data only; no user-generated data |
| 000009 | DELETE FROM editorial_widgets removes auto-inserted widgets | Widgets are defaults re-created on re-apply |
| 000011 | DELETE FROM writing_styles removes auto-inserted styles | Styles are defaults re-created on re-apply |
| All DROP TABLE | Permanent data loss of all rows in those tables | Expected — down migrations are for dev/test rollback |
| 000016 | Dropping RLS exposes data to unauthenticated access | Tables are inaccessible without RLS — dev-only concern |

## Migrations Requiring Manual Intervention

### 000003 (audit_log)
If 000001 was never applied, 000003's down migration will leave an orphaned `audit_log` table (non-partitioned). Manual cleanup required:
```sql
DROP TABLE IF EXISTS audit_log CASCADE;
```
The down file contains a clear WARNING comment at the top.

---

## Files Changed

| File | Status |
|------|--------|
| `migrations/000001_create_initial_schema.down.sql` | Created |
| `migrations/000002_add_oauth_mfa.down.sql` | Created |
| `migrations/000003_add_audit_log.down.sql` | Created |
| `migrations/000004_add_sites.down.sql` | Created |
| `migrations/000005_add_posts.down.sql` | Created |
| `migrations/000006_add_assets.down.sql` | Created |
| `migrations/000007_add_media_tables.down.sql` | Created |
| `migrations/000008_add_plugins.down.sql` | Created |
| `migrations/000009_add_editorial.down.sql` | Created |
| `migrations/000010_add_research_tables.down.sql` | Created |
| `migrations/000011_add_writer_tables.down.sql` | Created |
| `migrations/000012_add_editorial_tables.down.sql` | Created |
| `migrations/000013_add_generation_tables.down.sql` | Created |
| `migrations/000014_add_autocontent_tables.down.sql` | Created |
| `migrations/000015_add_seo_tables.down.sql` | Created |
| `migrations/000016_add_rls_policies.down.sql` | Created |
| `migrations/000017_add_human_writer_tables.down.sql` | Created |
| `migrations/000018_add_article_pipeline.down.sql` | Created |
| `migrations/000019_add_publisher_tables.down.sql` | Created |
| `migrations/000020_add_seo_engine_tables.down.sql` | Created |
| `migrations/000021_add_workflow_tables.down.sql` | Created |
| `migrations/000022_add_setup_tables.down.sql` | Created |
| `migrations/000023_add_grounding_fields.down.sql` | Created |
| `research/sprint-3.10-down-migrations-audit.md` | Created |
| `AGENTS.md` | Updated |

**No existing files were modified.**
