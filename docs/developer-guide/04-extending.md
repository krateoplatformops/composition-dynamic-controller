# Extending the CDC

Where to make the common changes. The CDC's logic is concentrated in a few places.

| You want to… | Change |
| --- | --- |
| Change reconcile behavior | The four operations (`Observe` / `Create` / `Update` / `Delete`) — the single business-logic seam the runtime calls. |
| Resolve chart coordinates differently | The chart resolver — provide a new source of a chart's location/version/credentials. A fixed-chart option already swaps in an alternative resolver. |
| Tighten the generated RBAC | The RBAC generator — it grants all verbs today; narrow it here. |
| Change labels or values on rendered objects | The post-renderer and value injection (in the shared Helm library). |
| Change which event a watched change triggers | The controller's launch options (for example, the update-becomes-observe remap). |
| Add a configuration knob | A new flag/env at startup. |

Keep the four operations **idempotent and non-blocking** — the runtime can re-enqueue and retry them, so a create must tolerate an already-existing release and a delete a missing one.

## Worked example: a new chart-coordinate source

Suppose chart coordinates should come from a ConfigMap rather than the `CompositionDefinition`. The shape of the change is: implement a new chart resolver that reads the ConfigMap referenced by the resource and produces the same chart-coordinate result the rest of the pipeline expects, then select it at startup (the same way the fixed-chart option is selected today). Nothing downstream changes, because the reconcile flow only depends on the resolved coordinates, not on where they came from.
