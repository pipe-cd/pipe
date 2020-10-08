---
title: "Feature Status"
linkTitle: "Feature Status"
weight: 8
description: >
  This page lists the relative maturity of every PipeCD features.
---

Please note that the phases (Incubating, Alpha, Beta, and Stable) are applied to individual features within the project, not to the project as a whole.

## Feature Phase Definitions

| Phase | Definition |
|-|-|
| Incubating | Under planning/developing the prototype and still not ready to be used. |
| Alpha | Demo-able, works end-to-end but has limitations. No guarantees on backward compatibility. |
| Beta | Usable in production. Documented. |
| Stable | Production hardened. Backward compatibility. Documented. |

## PipeCD Features

### Kubernetes Deployment

| Feature | Phase |
|-|-|
| Quick Sync Deployment | Alpha |
| Deployment with the Specified Pipeline (canary, bluegreen...) | Alpha |
| Automated Rollback | Alpha |
| [Automated Configuration Drift Detection](/docs/user-guide/configuration-drift-detection/) | Alpha |
| [Application Live State](/docs/user-guide/application-live-state/) | Alpha |
| Support Helm | Alpha |
| Support Kustomize | Alpha |
| Support Istio Mesh | Alpha |
| Support SMI Mesh | Incubating |

### Terraform Deployment

| Feature | Phase |
|-|-|
| Quick Sync Deployment | Alpha |
| Deployment with the Specified Pipeline | Alpha |
| Automated Rollback | Alpha |
| [Automated Configuration Drift Detection](/docs/user-guide/configuration-drift-detection/) | Incubating |
| [Application Live State](/docs/user-guide/application-live-state/) | Incubating |

### CloudRun Deployment

| Feature | Phase |
|-|-|
| Quick Sync Deployment | Alpha |
| Deployment with the Specified Pipeline | Alpha |
| Automated Rollback | Alpha |
| [Automated Configuration Drift Detection](/docs/user-guide/configuration-drift-detection/) | Incubating |
| [Application Live State](/docs/user-guide/application-live-state/) | Incubating |

### Lambda Deployment

| Feature | Phase |
|-|-|
| Quick Sync Deployment | Incubating |
| Deployment with the Specified Pipeline | Incubating |
| Automated Rollback | Incubating |
| [Automated Configuration Drift Detection](/docs/user-guide/configuration-drift-detection/) | Incubating |
| [Application Live State](/docs/user-guide/application-live-state/) | Incubating |

### Piped's Core

| Feature | Phase |
|-|-|
| [Wait Stage](/docs/user-guide/adding-a-wait-stage/) | Beta |
| [Wait Manual Approval Stage](/docs/user-guide/adding-a-manual-approval/) | Beta |
| [ADA](/docs/user-guide/automated-deployment-analysis/) (Automated Deployment Analysis) by Prometheus metrics | Alpha |
| [ADA](/docs/user-guide/automated-deployment-analysis/) by Datadog metrics | Incubating |
| [ADA](/docs/user-guide/automated-deployment-analysis/) by Stackdriver metrics | Incubating |
| [ADA](/docs/user-guide/automated-deployment-analysis/) by Stackdriver log | Incubating |
| [ADA](/docs/user-guide/automated-deployment-analysis/) by CloudWatch metrics | Incubating |
| [ADA](/docs/user-guide/automated-deployment-analysis/) by CloudWatch log | Incubating |
| [ADA](/docs/user-guide/automated-deployment-analysis/) by HTTP request (smoke test...) | Incubating |
| [Notification](/docs/operator-manual/piped/configuring-notifications/) to Slack | Beta |
| [Notification](/docs/operator-manual/piped/configuring-notifications/) to Webhook | Incubating |
| [Image Watcher](/docs/user-guide/image-watcher/) | Incubating |
| Secrets Management | Incubating |

### ControlPlane's Core

| Feature | Phase |
|-|-|
| Project/Environment/Piped/Application/Deployment Management | Beta |
| Rendering Deployment Pipeline in Realtime | Beta |
| Canceling a Deployment from Web | Beta |
| Triggering a Sync/Deployment from Web | Beta |
| Authentication by Username/Password for Static Admin | Beta |
| GitHub & GitHub Enterprise SSO | Beta |
| Google SSO | Incubating |
| Bitbucket SSO | Incubating |
| Support GCP [Firestore](https://cloud.google.com/firestore) as a data store of the control plane | Beta |
| Support AWS [DynamoDB](https://aws.amazon.com/dynamodb/) as a data store of the control plane | Incubating |
| Support [MongoDB](https://www.mongodb.com/) as a data store of the control plane | Alpha |
| Support GCP [GCS](https://cloud.google.com/storage) as a file store of the control plane | Beta |
| Support AWS [S3](https://aws.amazon.com/s3/) as a file store of the control plane | Incubating |
| Support [Minio](https://github.com/minio/minio) as a file store of the control plane | Alpha |
| [Insights](/docs/user-guide/insights/) shows delivery performance | Incubating |
| Collecting piped's metrics and enabling their dashboards | Incubating |
