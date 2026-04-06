# Data Model: Internal Test Deployment Pipeline

## Entity: DeploymentTask

- **Purpose**: Represents one actionable setup/release step.
- **Fields**:
  - `id` (string, required) - stable identifier (e.g., `android.play-console.create-app`)
  - `title` (string, required)
  - `description` (string, required)
  - `platform` (enum: `android` | `ios` | `shared`, required)
  - `category` (enum: `account`, `credentials`, `eas-config`, `ci`, `ota`, `validation`, required)
  - `status` (enum: `todo` | `in_progress` | `blocked` | `done`, required)
  - `owner_role` (string, required) - e.g., `mobile-engineer`, `devops`, `org-admin`
  - `blocking` (boolean, required)
  - `dependency_ids` (string[], default `[]`)
  - `artifacts` (string[], default `[]`) - repo paths or console outputs
  - `notes` (string, optional)
- **Validation Rules**:
  - `id` unique across list.
  - If `blocking = true`, task must have at least one dependent downstream task.
  - `dependency_ids` must reference existing task IDs.

## Entity: CredentialAsset

- **Purpose**: Tracks signing/push credential assets and storage locations.
- **Fields**:
  - `id` (string, required)
  - `type` (enum: `android_keystore`, `fcm_service_account`, `apns_key`, `apple_profile`, `eas_token`, required)
  - `provider` (enum: `google`, `apple`, `expo`, `github`, required)
  - `created_by` (string, required)
  - `rotation_policy_days` (number, optional)
  - `backup_location` (string, required)
  - `last_verified_at` (datetime, optional)
  - `scope` (enum: `development`, `staging`, `production`, `shared`, required)
  - `active` (boolean, required)
- **Validation Rules**:
  - `backup_location` cannot be local personal disk.
  - `active` credential must have `last_verified_at` within defined rotation window.

### Credential Asset Execution Checklist

- For each `CredentialAsset`, capture:
  - custody owner (`created_by`) and backup owner (secondary operator)
  - secret storage locations (GitHub Environments, Expo Secrets, vault path)
  - restoration drill timestamp and result (`last_verified_at`)
  - rotation interval (`rotation_policy_days`) and next due date
- Critical credential types that must exist before release:
  - `android_keystore`
  - `apns_key`
  - `eas_token`
  - `fcm_service_account` (or equivalent delegated credential model)

## Entity: BuildProfile

- **Purpose**: Defines EAS build profile behavior.
- **Fields**:
  - `name` (enum: `development`, `preview`, `production`, required)
  - `distribution` (enum: `internal`, `store`, required)
  - `development_client` (boolean, required)
  - `channel` (string, required)
  - `auto_increment` (boolean|string, optional)
  - `env_set` (string, required)
- **Validation Rules**:
  - `development` must set `development_client=true`.
  - `preview` should use `distribution=internal`.
  - `production` must target store-ready config.

## Entity: SubmissionRun

- **Purpose**: Records one build+submit execution per platform/profile.
- **Fields**:
  - `id` (string, required)
  - `platform` (enum: `android` | `ios`, required)
  - `profile` (enum: `preview` | `production`, required)
  - `build_id` (string, required)
  - `submit_id` (string, optional)
  - `trigger` (enum: `manual`, `workflow_dispatch`, `push`, required)
  - `status` (enum: `queued`, `building`, `submitted`, `failed`, `approved`, required)
  - `started_at` (datetime, required)
  - `finished_at` (datetime, optional)
  - `log_url` (string, optional)
- **State Transitions**:
  - `queued -> building -> submitted -> approved`
  - `queued|building|submitted -> failed`

## Entity: PushEnvironment

- **Purpose**: Captures end-to-end push readiness per platform.
- **Fields**:
  - `platform` (enum: `android` | `ios`, required)
  - `expo_project_id` (string, required)
  - `native_provider` (enum: `fcm` | `apns`, required)
  - `native_config_ready` (boolean, required)
  - `eas_credentials_ready` (boolean, required)
  - `smoke_test_passed` (boolean, required)
  - `last_tested_at` (datetime, optional)
- **Validation Rules**:
  - `smoke_test_passed=true` requires both readiness flags true.

## Relationships

- `DeploymentTask.dependency_ids` creates a DAG across tasks.
- `BuildProfile.channel` maps to OTA channels and influences `SubmissionRun`.
- `CredentialAsset` entries support both `BuildProfile` and `PushEnvironment`.
- Push validation tasks require matching `PushEnvironment` readiness.
