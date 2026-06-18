# Reconcile lifecycle

What happens each time a `Composition` is reconciled — the Helm flow, drift detection — and the one behavior you must understand before changing anything: **`Observe` mutates the cluster**.

The runtime calls the four operations; what `Observe` reports decides whether a create or update follows. Because the CDC treats an update as an observe (see [`01`](./01-architecture.md)), almost every event lands in `Observe`.

## Observe — the center of gravity

In order, `Observe`:

1. **Computes and persists the release name** (sticky once set; by default it gets a unique suffix).
2. **Honors graceful pause** — if the resource is paused, it returns immediately as "exists and up to date".
3. **Resolves the chart** and records the `CompositionDefinition` labels on the resource.
4. **Looks at the existing Helm release.** If there is none, it reports "doesn't exist" (which triggers a create). If the release is stuck in a pending state, it rolls back to the previous revision before continuing.
5. **Scopes RBAC** — asks chart-inspector which resources the chart touches and applies the corresponding roles (see [`03-rbac-generation.md`](./03-rbac-generation.md)).
6. **Builds the values** from the resource's spec, injects the global values, and prepares a post-renderer that stamps composition-ownership labels onto every rendered object.
7. **Runs a Helm upgrade.** This actually changes the cluster on every reconcile — see the box below.
8. **Detects drift** by computing a digest of the rendered release and comparing it to what was recorded. If it changed (or the chart version changed), it reports "out of date" (which triggers an update); otherwise it marks the composition available.

> ### ⚠️ `Observe` mutates the cluster
> The Helm upgrade in step 7 runs **unconditionally, on every reconcile, before** the drift comparison. The practical effect is roughly one Helm revision per reconcile, plus a steady stream of "up-to-date" logs. It is intentional today — Helm's three-way merge quietly corrects manual drift on the live objects as a side effect — but it is the single biggest footgun in this codebase. The release-history cap limits how many revisions are *kept*, not the churn. Read `docs/observe-reconciliation-loop-184.md` before touching `Observe`.

## Create, Update, Delete

- **Create** installs the chart (or upgrades an existing release), records the resulting objects in the status, and marks the composition available.
- **Update** re-reads the release, recomputes the digest and the recorded objects, and updates status.
- **Delete** uninstalls the release, confirms it is gone, then removes the generated RBAC. Finalizer handling is the runtime's job, not the handler's.

## The Helm flow

All Helm operations (get, install, upgrade, uninstall, rollback) go through the shared Helm library; the actual rendering and apply happen inside it. The CDC contributes two things: the **values** (the resource's spec plus a `global` block carrying the composition's identity and the Krateo namespace) and a **post-renderer** that labels every rendered object with composition ownership. Changing the labels or values applied to rendered objects is done at that post-renderer.

## Drift detection

Drift is detected by hashing the **rendered** manifest and comparing it to the recorded digest. This is rendered-vs-rendered, not rendered-vs-live: it does not directly notice edits to live objects — only the side-effecting upgrade (see the box) corrects those.

## Creating a composition for resources that already exist (adoption)

Two distinct "already exists" questions come up, and they have different answers.

**The Helm release already exists.** `Observe` (and `Create`) look up the release by its computed name first. If a release is already there — typically because a previous `Create` failed *after* installing, or the controller restarted — the CDC **upgrades the existing release instead of installing a new one**, so a re-triggered create is safe and idempotent rather than a "release already exists" failure.

**A Kubernetes object the chart renders already exists in the cluster.** Here the CDC relies on Helm's default ownership rules. The post-renderer stamps every rendered object with the composition-ownership labels, but the CDC does **not** enable Helm's take-ownership / force / replace behavior. So if the chart would create an object that already exists and is *not* part of this release, Helm refuses to adopt it and the install/upgrade fails with the usual *"exists and cannot be imported into the current release"* / invalid-ownership error. Adopting pre-existing objects would require Helm's take-ownership option — it exists in the shared Helm library but is not wired into the CDC today, so importing arbitrary live objects into a composition is **not** supported out of the box.

## Disabling specific operations (management & deletion policies)

The CDC runs on unstructured-runtime, which honors two annotations on the **`Composition` instance** to switch off individual operations — the same annotations core-provider honors on the `CompositionDefinition`:

| Annotation | Value | What the CDC may do to the release |
| --- | --- | --- |
| `krateo.io/management-policy` | `default` (when unset) | Full management: create, update, and delete the release. |
| | `observe-create-update` | Create and update, but **never uninstall** on delete. |
| | `observe-delete` | **No create/update**; only delete (uninstall). |
| | `observe` | **Observe only** — never create, update, or delete. |
| `krateo.io/deletion-policy` | `delete` (when unset) / `orphan` | With `orphan`, deleting the `Composition` removes the finalizer **without** uninstalling the release. |

Mechanically these gate the create/update/delete events the reconcile loop would otherwise enqueue (`meta.ShouldCreate` / `ShouldUpdate` / `ShouldDelete` in unstructured-runtime). User-facing usage and YAML examples are not duplicated here — see the **Lifecycle Policies** how-to on [docs.krateo.io](https://docs.krateo.io).

> ### ⚠️ `observe` / `observe-create-update` do not fully freeze an existing release
> These policies suppress the **enqueued** create/update actions — but remember that **`Observe` itself runs a Helm upgrade** on every reconcile when a release already exists (see the box near the top of this page). The management policy does *not* gate that side-effecting upgrade, so a composition whose release already exists is **not** frozen by `observe`; the upgrade inside `Observe` still runs. (Create is genuinely prevented, because `Observe` reports "doesn't exist" and returns before upgrading when there is no release.) **To truly freeze a composition, use graceful pause**, which returns early from all four operations.

## Graceful pause

Graceful pause is a CDC-specific signal, distinct from the runtime's standard pause. When set, all four operations return early without touching the release, and the paused duration is tracked. It's how a composition can be frozen without being deleted.
