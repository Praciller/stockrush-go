# Hosting Evaluation

Accessed: 2026-07-13. Only official provider documentation was used. Provider terms can change; re-check these sources before any later redeployment.

## Decision

The only currently identified combination that meets the stated no-card, no-trial, no-automatic-billing constraint is:

- Static frontend: GitHub Pages from this public repository.
- Go API and expiration worker: one public Hugging Face Docker Space on CPU Basic, with the worker co-located with the API because the free plan exposes one container.
- PostgreSQL: Neon Free plan over TLS.

This is a portfolio demo architecture, not an availability-grade production platform. The Space sleeps after inactivity and Neon compute scales to zero. The frontend must retain deterministic static evidence during cold starts or outages.

Production hardening and local verification are complete. Deployment is blocked by database account provisioning: Neon project creation returned HTTP 404 in both the available Vercel-managed free organization and the provider default path, while the Supabase fallback connector requires reauthentication before its organization, cost, and card requirements can be verified.

## Deployment Attempt

Attempted on 2026-07-13 after the local verification gate passed:

- Hugging Face authentication succeeded as `Pracill`; no paid hardware or Space was created because no database was available.
- Neon authentication succeeded and exposed one free Vercel-managed organization, but both scoped and default `stockrush-go` project creation returned HTTP 404. No Neon project or billable resource was created.
- Supabase Free was checked as a fallback. Official pricing documents `$0/month`, two active projects, 500 MB database size, 5 GB egress, no automatic backups, restriction instead of billed Free-plan overage, and pause after one week of inactivity. The connector required reauthentication, and the reviewed official documentation did not explicitly confirm card-free project creation. No Supabase project was created.
- GitHub Pages remains unchanged and continues to be the only verified public surface.

The live frontend/API switch, cloud secret writes, and public verification were not attempted after the database provisioning gate failed.

## Provider Matrix

| Provider / role | Free status | Card | Compute / storage | Egress / bandwidth / builds | Sleep / inactivity / deletion | TLS / domains / database network | Backup | Verdict |
|---|---|---|---|---|---|---|---|---|
| GitHub Pages / static frontend | Included for public repositories on GitHub Free; not a trial | No card required for GitHub Free public Pages | Static hosting; repository and Actions limits apply | Public-repository standard Actions runners are free; Pages is static only | Site follows repository/account lifecycle | Managed HTTPS; custom domains supported | Git history and build artifacts, not application data | Accept |
| Hugging Face Spaces CPU Basic / Go API + worker | CPU Basic is documented as free; a card or grant is required only to upgrade hardware | No card required for CPU Basic; do not add a payment method or request upgraded hardware | 2 vCPU, 16 GB RAM, 50 GB non-persistent disk; public source; one externally exposed app port | Official reviewed docs do not publish a fixed application-bandwidth or build-minute allowance; Hub action limits are platform-controlled. No paid hardware is selected, so exhaustion can suspend/throttle but cannot create a charge without an upgrade/payment method | Free Space sleeps after about 48 hours of inactivity and wakes on a visitor; local disk is ephemeral; free CPU cannot disable sleep | Managed public HTTPS; outbound connections allowed on 80/443/8080; Docker Space secrets are runtime environment variables | No persistent local backup; database is external | Accept with cold-start and single-container limitations |
| Neon Free / PostgreSQL | `$0`, no time limit, no credit card | Explicitly no card required | 100 CU-hours/month per project, 0.5 GB storage/project, compute up to 2 CU; pooled connections supported | 5 GB/month public network transfer; exceeding it suspends compute until the next cycle or upgrade, rather than charging the Free plan | Compute scales to zero after 5 minutes idle; inactive branches may be archived; reviewed docs do not state automatic project deletion for ordinary inactivity | TLS connection endpoint; public network endpoint rather than a private network on Free; application role and strong credentials required | Six-hour time-travel/restore window; operator `pg_dump` is still required | Accept with tight storage/egress/restore limits |
| Supabase Free / PostgreSQL fallback | `$0/month`, not documented as a trial | Card-free creation was not explicit in reviewed official docs and could not be checked because connector reauthentication is required | Two active free projects; 500 MB database; shared CPU and 500 MB RAM | 5 GB egress; Free-plan overage is restricted rather than billed | Paused after one week of inactivity | Managed TLS endpoint; direct and pooled PostgreSQL connectivity | Automatic backups are not included | Reject for this run: account/card gate could not be verified |
| Render Free / API or PostgreSQL | Free web services exist, but free PostgreSQL expires after 30 days | Pricing documents cards for paid services; card-free signup was not sufficiently explicit in reviewed docs | Web: 512 MB, 0.1 CPU; Postgres: 1 GB and one instance | 750 free instance-hours; 5 GB included bandwidth, then `$0.15/GB`; service-to-external-DB traffic counts | Web sleeps after 15 minutes; Postgres expires after 30 days and is deleted after grace period | Managed TLS; free web cannot receive private-network traffic | Free database is temporary | Reject: time-limited database and possible bandwidth billing |
| Koyeb Free / API or worker | One free web service and a very limited free PostgreSQL offer | Credit card required with a `$29` pre-authorization; spending limits are not available | One 512 MB / 0.1 vCPU / 2 GB web instance; free Postgres limited to 5 active hours and 1 GB | 100 GB outbound currently free, with announced future paid overage | Free web scales to zero after 1 hour | Managed TLS; free instance cannot be a Worker Service | Not sufficient for this requirement | Reject: card required, no free worker, database active-time limit |
| Railway Free / full stack | New accounts start with a 30-day `$5` credit trial; subsequent Free plan has `$1` monthly credit | Trial itself says no card, but paid usage uses cards | 0.5 GB RAM/service after trial; one project; resource use consumes credit | Metered compute and `$0.05/GB` egress; service suspension follows credit exhaustion | Trial expires after 30 days | Domains/private networking are reduced after trial | Feature list includes volume backups, but compute remains credit-metered | Reject: onboarding relies on a time-limited promotional trial and credits |

## Candidate Limits and Controls

### GitHub Pages

- Use the existing verified Pages workflow and public repository.
- Build only public API origin configuration into the frontend.
- Never build API keys, database URLs, or operator controls into frontend assets.

Official sources:

- <https://docs.github.com/en/get-started/learning-about-github/githubs-plans>
- <https://docs.github.com/en/pages/getting-started-with-github-pages/what-is-github-pages>

### Hugging Face Docker Space

- Select only `cpu-basic`; never request upgraded hardware.
- Do not add a payment card.
- Run the API and expiration loop in the same container because one free public container is the available compute boundary.
- Bind to the Space `app_port` and store secrets only in Space Secrets.
- Expect a roughly 48-hour idle sleep, cold start, restart on deployment, and loss of local filesystem state.
- The reviewed official docs do not quantify application bandwidth or build-minute quotas. This is an availability uncertainty, not a billing path while only free CPU is selected without a card.

Official sources:

- <https://huggingface.co/docs/hub/main/spaces-overview>
- <https://huggingface.co/docs/hub/main/spaces-sdks-docker>
- <https://huggingface.co/docs/hub/spaces-gpus>
- <https://huggingface.co/docs/huggingface_hub/guides/manage-spaces>

### Neon Free PostgreSQL

- Use a TLS connection string and the pooled endpoint where appropriate.
- Create a least-privilege application role and keep migration privileges separate when practical.
- Cap the application pool well below the provider connection limit.
- Budget for 100 CU-hours, 0.5 GB storage, and 5 GB monthly egress.
- Expect five-minute compute scale-to-zero and a wake on the next query.
- The six-hour restore window is not a substitute for an operator logical backup.

Official sources:

- <https://neon.com/pricing>
- <https://neon.com/docs/introduction/network-transfer>
- <https://neon.com/docs/introduction/scale-to-zero>
- <https://neon.com/docs/connect/connection-pooling>

### Rejected Providers

- Supabase: <https://supabase.com/pricing>, <https://supabase.com/docs/guides/platform/billing-faq>, <https://supabase.com/docs/guides/platform/database-size>
- Render: <https://render.com/docs/free>, <https://render.com/pricing>
- Koyeb: <https://www.koyeb.com/docs/faqs/pricing>, <https://www.koyeb.com/docs/reference/instances>
- Railway: <https://railway.com/pricing>

## Cost Safety

- No payment card may be added for this deployment.
- No paid hardware, paid database plan, persistent disk, replica, or add-on may be selected.
- No service may be configured to auto-upgrade.
- Public load testing is prohibited; full correctness and soak tests remain local.
- If either provider requires a card, trial, paid upgrade, or usage billing during account setup, stop deployment and retain Level A.

## Data and Deletion

- All rows are synthetic and may be deleted without customer impact.
- The Space filesystem is disposable and contains no authoritative state.
- Neon is authoritative only for the portfolio demo and must stay below documented Free plan limits.
- Operator logical backups are local, ignored by Git, and periodically tested through a fresh-database restore drill.
