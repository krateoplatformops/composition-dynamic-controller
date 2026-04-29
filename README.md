# Composition Dynamic Controller (CDC)

The **Composition Dynamic Controller (CDC)** is the execution engine of Krateo. It is a specialized operator that manages the full lifecycle of Helm-based services by watching and reconciling `Composition` resources.

## Key Features

- **Lifecycle Orchestration**: Manages the end-to-end deployment, updates, and deletion of services based on Helm charts.
- **Dynamic Reconciliation**: Automatically reconciles resource states, ensuring the live cluster matches the desired state defined in the `Composition`.
- **Chart Inspector Integration**: Leverages the Krateo Chart Inspector for secure dry-runs, ensuring chart validity and resource safety before application.

## Security & Operational Design

- **RBAC Enforcement**: Provisions specific, fine-grained RBAC policies for each managed composition, enforcing least-privilege principles at the instance level.
- **Graceful Lifecycle Management**: Supports advanced management features like service pausing/resuming and controlled Helm release versioning.
- **Observability**: Built-in support for OpenTelemetry to monitor reconciliation health and performance metrics.

## Documentation

For detailed guides, architecture diagrams, and full reference, visit the official documentation:
👉 **[https://docs.krateo.io](https://docs.krateo.io/key-concepts/kco/cdc/overview)**
