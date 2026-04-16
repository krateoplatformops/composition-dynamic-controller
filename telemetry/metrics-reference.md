# Unstructured-Runtime Metrics Reference

This document describes the OpenTelemetry metrics emitted by `unstructured-runtime` and `composition-dynamic-controller`.

## Naming note

Metric names in code use dots. Prometheus normalizes them with underscores, and counters appear with a `_total` suffix.
Histogram queries use the generated `_bucket` series (cumulative count), and `_sum` / `_count` series for average duration.
For average duration, use `_sum` divided by `_count` instead of `_bucket`.

## Core Metrics

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.startup.success` | Counter | count | Controller started successfully. | `sum(increase(unstructured_runtime_startup_success_total[1h]))` |
| `unstructured_runtime.startup.failure` | Counter | count | Controller startup failed. | `sum(increase(unstructured_runtime_startup_failure_total[1h]))` |
| `unstructured_runtime.reconcile.duration_seconds` | Histogram | seconds | Total reconciliation duration. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_reconcile_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.reconcile.in_flight` | Gauge | count | Number of reconciliations currently in progress. | `max(unstructured_runtime_reconcile_in_flight)` |

## Queue Metrics

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.reconcile.queue.depth` | UpDownCounter | count | Current number of items in the queue. | `max(unstructured_runtime_reconcile_queue_depth)` |
| `unstructured_runtime.reconcile.queue.wait.duration_seconds` | Histogram | seconds | Time spent waiting in queue before processing. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_reconcile_queue_wait_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.reconcile.queue.oldest_item_age_seconds` | Histogram | seconds | Age of the oldest item currently in queue. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_reconcile_queue_oldest_item_age_seconds_bucket[5m])))` |
| `unstructured_runtime.reconcile.queue.work.duration_seconds` | Histogram | seconds | Time spent processing a dequeued item. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_reconcile_queue_work_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.reconcile.queue.requeues` | Counter | count | Total number of items requeued. | `sum(increase(unstructured_runtime_reconcile_queue_requeues_total[1h]))` |

## Reconciliation Results

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.reconcile.success` | Counter | count | Successfully completed reconciliations. | `sum(increase(unstructured_runtime_reconcile_success_total[1h]))` |
| `unstructured_runtime.reconcile.failure` | Counter | count | Failed reconciliations. | `sum(increase(unstructured_runtime_reconcile_failure_total[1h]))` |
| `unstructured_runtime.reconcile.requeue.after` | Counter | count | Reconciliations that requested requeue after delay. | `sum(increase(unstructured_runtime_reconcile_requeue_after_total[1h]))` |
| `unstructured_runtime.reconcile.requeue.immediate` | Counter | count | Reconciliations that requested immediate requeue. | `sum(increase(unstructured_runtime_reconcile_requeue_immediate_total[1h]))` |
| `unstructured_runtime.reconcile.requeue.error` | Counter | count | Reconciliations requeued due to error. | `sum(increase(unstructured_runtime_reconcile_requeue_error_total[1h]))` |

## Get Operation Metrics

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.reconcile.get.duration_seconds` | Histogram | seconds | Time to fetch/get resources. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_reconcile_get_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.reconcile.get.failure` | Counter | count | Failed get operations. | `sum(increase(unstructured_runtime_reconcile_get_failure_total[1h]))` |

## External Operation Metrics

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.external.observe.duration_seconds` | Histogram | seconds | Time to observe external resources. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_external_observe_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.external.observe.failure` | Counter | count | Failed observe operations. | `sum(increase(unstructured_runtime_external_observe_failure_total[1h]))` |
| `unstructured_runtime.external.connect.duration_seconds` | Histogram | seconds | Time to connect/read external references. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_external_connect_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.external.connect.failure` | Counter | count | Failed connect operations. | `sum(increase(unstructured_runtime_external_connect_failure_total[1h]))` |
| `unstructured_runtime.external.create.duration_seconds` | Histogram | seconds | Time to create external resources. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_external_create_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.external.create.failure` | Counter | count | Failed create operations. | `sum(increase(unstructured_runtime_external_create_failure_total[1h]))` |
| `unstructured_runtime.external.update.duration_seconds` | Histogram | seconds | Time to update external resources. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_external_update_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.external.update.failure` | Counter | count | Failed update operations. | `sum(increase(unstructured_runtime_external_update_failure_total[1h]))` |
| `unstructured_runtime.external.delete.duration_seconds` | Histogram | seconds | Time to delete external resources. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_external_delete_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.external.delete.failure` | Counter | count | Failed delete operations. | `sum(increase(unstructured_runtime_external_delete_failure_total[1h]))` |

## Finalizer Operation Metrics

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.finalizer.add.duration_seconds` | Histogram | seconds | Time to add finalizers. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_finalizer_add_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.finalizer.add.failure` | Counter | count | Failed add finalizer operations. | `sum(increase(unstructured_runtime_finalizer_add_failure_total[1h]))` |
| `unstructured_runtime.finalizer.remove.duration_seconds` | Histogram | seconds | Time to remove finalizers. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_finalizer_remove_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.finalizer.remove.failure` | Counter | count | Failed remove finalizer operations. | `sum(increase(unstructured_runtime_finalizer_remove_failure_total[1h]))` |

## Status Update Metrics

| Metric | Type | Unit | Description | PromQL example |
|---|---|---|---|---|
| `unstructured_runtime.status.update.duration_seconds` | Histogram | seconds | Time to update resource status. | `histogram_quantile(0.95, sum by (le) (rate(unstructured_runtime_status_update_duration_seconds_bucket[5m])))` |
| `unstructured_runtime.status.update.failure` | Counter | count | Failed status update operations. | `sum(increase(unstructured_runtime_status_update_failure_total[1h]))` |

## Setup Notes

- Metrics are tagged with the following resource attributes:
  - `service.name`: Application identifier
  - `k8s.deployment.name`: Kubernetes deployment name (for multi-instance identification)
  - `service.instance.id`: Stable instance identifier (set to deployment name)
- All metrics are low-cardinality
- Metrics are exported via OTLP HTTP to the OpenTelemetry Collector
- Prometheus scrapes the Collector's Prometheus exporter endpoint
- Grafana connects to Prometheus to visualize the metrics
- Histograms are configured with auto-generated bounds optimized for latency observation
