# Composition Dynamic Controller (CDC)

The **Composition Dynamic Controller (CDC)** is the execution engine of Krateo. It is a specialized operator that manages the full lifecycle of Helm-based services by watching and reconciling `Composition` resources.

## Key Features

- **Lifecycle Management**: Orchestrates the deployment, update, and deletion of Helm-based services.
- **Safety**: Integrates with the Chart Inspector to perform dry-runs and enable precise RBAC auto-provisioning.
- **Dynamic**: Instantiated by the Core Provider to handle specific `CompositionDefinition` instances.

## Documentation

For detailed guides, architecture diagrams, and full reference, visit the official documentation:

👉 **[https://docs.krateo.io](https://docs.krateo.io/key-concepts/kco/cdc/overview)**
