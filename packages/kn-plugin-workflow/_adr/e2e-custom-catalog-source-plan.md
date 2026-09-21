# E2E Custom CatalogSource Operator Install Plan

## Overview

The E2E tests for `kn-plugin-workflow` currently install the SonataFlow Operator from a public OLM catalog
(`operatorhubio-catalog` on Kubernetes, `community-operators` on OpenShift). This means E2E tests always
exercise the publicly released operator version and have no way to verify a specific internal build.

This change adds support for installing the operator from a **custom OLM CatalogSource** when three
environment variables are provided. The `CATALOG_INDEX_IMAGE` env var is the trigger: when it is set, the
E2E setup creates a custom `CatalogSource` pointing to that index image and subscribes to it instead of the
public catalog. `OPERATOR_BUNDLE_IMAGE` and `OPERATOR_IMAGE` are accepted alongside it and printed to CI
output for traceability — they are not used programmatically because they are already embedded in the index.

When none of the env vars are set, all existing behavior is preserved exactly.

Additionally, the Subscription channel is currently hardcoded as `"alpha"` but the current release of both
`sonataflow-operator` and `logic-operator` (product name) uses the `"stable"` channel. This will be
corrected as part of this task.

---

## Key Design Decisions

| Condition                         | Operator name in Subscription | Channel  | CatalogSource                                                             |
| --------------------------------- | ----------------------------- | -------- | ------------------------------------------------------------------------- |
| `CATALOG_INDEX_IMAGE` **not set** | `sonataflow-operator`         | `stable` | `operatorhubio-catalog` (Kubernetes) or `community-operators` (OpenShift) |
| `CATALOG_INDEX_IMAGE` **set**     | `logic-operator`              | `stable` | custom `osl-catalog` in `olm` namespace                                   |

- Custom-catalog mode always implies a product/OSL build, so `logic-operator` is used unconditionally when the index image is provided — no extra env var needed.
- `OPERATOR_BUNDLE_IMAGE` and `OPERATOR_IMAGE` are logged to CI output for traceability only.

---

## Architecture Summary

```
Custom mode (CATALOG_INDEX_IMAGE set):
  env vars → operator_helper.go → OperatorManager
    → CreateCustomCatalogSource("osl-catalog", "olm", indexImage)
    → waitForCatalogSourceReady("osl-catalog", "olm")
    → InstallOperatorFromCatalog("logic-operator", "stable", "osl-catalog", "olm")
    → [on uninstall] RemoveCustomCatalogSource("osl-catalog", "olm")

Default mode (no env vars):
  [unchanged — uses operatorhubio-catalog / community-operators, channel "stable"]
```

---

## Sub-Tasks

---

### Sub-Task 1 — Fix the Subscription channel from `"alpha"` to `"stable"`

**Status:** `[x] done`

**Intent:**
The current `InstallSonataflowOperator()` in [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go)
hardcodes `"channel": "alpha"` in the OLM Subscription spec. The `sonataflow-operator` v10.1.0 and the
product `logic-operator` both publish to the `"stable"` channel. This is a correctness fix that applies
to all install paths, independent of the custom-catalog feature.

**Expected Outcomes:**

- The Subscription created by `InstallSonataflowOperator()` uses `"channel": "stable"`.
- No other behaviour changes.

**Todo List:**

1. In [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) line 226, change the
   channel value from `"alpha"` to `"stable"`.

**Relevant Context:**

- [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) line 226 — `"channel": "alpha"`.

---

### Sub-Task 2 — Read and expose the three image environment variables in test setup

**Status:** `[x] done`

**Intent:**
Centralise the reading of `CATALOG_INDEX_IMAGE`, `OPERATOR_BUNDLE_IMAGE`, and `OPERATOR_IMAGE` environment
variables into the E2E test layer. This gives a single place to see whether custom-catalog mode is active,
and emits clear log lines with the values so CI output is traceable.

**Expected Outcomes:**

- Three package-level variables are declared in the E2E test package to hold the env var values.
- `TestMain` reads them at startup, logs their values (or `"not set"`), and logs which install mode will be used.
- No test behaviour changes yet — the variables exist but nothing branches on them.

**Todo List:**

1. In [`e2e-tests/test_env.go`](packages/kn-plugin-workflow/e2e-tests/test_env.go) add three exported string
   variables: `CatalogIndexImage`, `OperatorBundleImage`, `OperatorImage`.
2. In [`e2e-tests/main_test.go`](packages/kn-plugin-workflow/e2e-tests/main_test.go), inside `TestMain`,
   after the executable is built, read the three env vars into those variables using `os.Getenv`.
3. Log each variable value (or `"not set"`) using `fmt.Println` so they appear in CI output.
4. Log one summary line: `"🔧 Custom catalog mode: enabled (product build)"` or
   `"🔧 Custom catalog mode: disabled (using public catalog)"`.

**Relevant Context:**

- [`e2e-tests/test_env.go`](packages/kn-plugin-workflow/e2e-tests/test_env.go) — existing package-level vars.
- [`e2e-tests/main_test.go`](packages/kn-plugin-workflow/e2e-tests/main_test.go) — `TestMain` entry point.

---

### Sub-Task 3 — Add `CreateCustomCatalogSource` and `RemoveCustomCatalogSource` to `OperatorManager`

**Status:** `[x] done`

**Intent:**
Extend [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) with two new methods
on `OperatorManager`:

- `CreateCustomCatalogSource(name, namespace, indexImage string) error` — creates an OLM `CatalogSource`
  resource of type `grpc` pointing to the provided index image.
- `RemoveCustomCatalogSource(name, namespace string) error` — deletes that `CatalogSource` by name/namespace.

**Expected Outcomes:**

- The `OperatorManager` can create a `CatalogSource` of kind `operators.coreos.com/v1alpha1` in any given
  namespace using a caller-supplied index image.
- The same manager can delete it by name and namespace.
- No existing methods are modified.

**Todo List:**

1. In [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) add
   `CreateCustomCatalogSource(name, namespace, indexImage string) error`:
   - Build an `unstructured.Unstructured` for a `CatalogSource` with `spec.sourceType: "grpc"` and
     `spec.image` set to `indexImage`.
   - Use the existing `ExecuteCreate(catalogSourcesGVR, ...)` helper to apply it.
2. Add `RemoveCustomCatalogSource(name, namespace string) error`:
   - Use the existing `ExecuteDeleteGVR(catalogSourcesGVR, name, namespace)` helper.

**Relevant Context:**

- [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) — `catalogSourcesGVR` is
  already declared; `ExecuteCreate` and `ExecuteDeleteGVR` helpers already exist.
- The `CatalogSource` manifest shape follows the OLM API `operators.coreos.com/v1alpha1`.

---

### Sub-Task 4 — Extend `InstallSonataflowOperator` to accept a custom source and operator name

**Status:** `[x] done`

**Intent:**
Modify [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) so that the
Subscription creation logic can be called with a caller-supplied operator name, channel, catalog source, and
source namespace. This is needed because the product operator is named `logic-operator` (not
`sonataflow-operator`) and points to `osl-catalog`.

The existing `InstallSonataflowOperator()` public method must remain unchanged in behaviour for all existing
callers — it continues to use `sonataflow-operator` / `stable` / `operatorhubio-catalog` or
`community-operators`.

**Expected Outcomes:**

- A new internal method `installOperatorWithSource(operatorName, channel, source, sourceNamespace string) error`
  centralises the Subscription construction.
- `InstallSonataflowOperator()` delegates to it with the existing default values (updated channel: `stable`).
- A new public method `InstallOperatorFromCatalog(operatorName, channel, source, sourceNamespace string) error`
  delegates directly to the internal method — for use by the E2E custom-catalog path.

**Todo List:**

1. Extract `installOperatorWithSource(operatorName, channel, source, sourceNamespace string) error` from the
   body of `InstallSonataflowOperator()`.
2. `InstallSonataflowOperator()` calls `installOperatorWithSource` with:
   - `operatorName` = `metadata.SonataFlowOperatorName` (`"sonataflow-operator"`)
   - `channel` = `"stable"`
   - `source` / `sourceNamespace` = the existing platform-specific defaults.
3. Add public `InstallOperatorFromCatalog(operatorName, channel, source, sourceNamespace string) error`
   that calls `installOperatorWithSource` directly.

**Relevant Context:**

- [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) lines 205–241 —
  `InstallSonataflowOperator`.
- [`pkg/metadata/constants.go`](packages/kn-plugin-workflow/pkg/metadata/constants.go) — `SonataFlowOperatorName`.
  A new constant `LogicOperatorName = "logic-operator"` should be added here.

---

### Sub-Task 5 — Update `CheckOLMInstalled` and `OLMCatalogSourcesMap` to accept the custom catalog

**Status:** `[x] done`

**Intent:**
When a custom `CatalogSource` named `osl-catalog` is created in the `olm` namespace, the existing
`CheckOLMInstalled()` fails because it only looks for `operatorhubio-catalog` or OpenShift sources.
Add `osl-catalog` to the known sources map so the OLM health check succeeds in custom-catalog mode.

**Expected Outcomes:**

- `"osl-catalog"` in namespace `"olm"` is recognised as a valid OLM installation by `CheckOLMInstalled`.
- Default mode behaviour is unchanged.

**Todo List:**

1. In [`pkg/metadata/constants.go`](packages/kn-plugin-workflow/pkg/metadata/constants.go), add
   `"osl-catalog": "olm"` to `OLMCatalogSourcesMap`.
2. Add a new constant `LogicOperatorName = "logic-operator"` alongside `SonataFlowOperatorName`.
3. Verify that `CheckOLMInstalled()` in [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go)
   naturally picks up the new entry — no code change needed there since it already iterates the map.

**Relevant Context:**

- [`pkg/metadata/constants.go`](packages/kn-plugin-workflow/pkg/metadata/constants.go) lines 57–59 —
  `OLMCatalogSourcesMap` and `SonataFlowOperatorName`.
- [`pkg/common/operator.go`](packages/kn-plugin-workflow/pkg/common/operator.go) lines 176–203 —
  `CheckOLMInstalled` iterates `OLMCatalogSourcesMap` entries.

---

### Sub-Task 6 — Wire everything together in the E2E `operator_helper.go`

**Status:** `[x] done`

**Intent:**
Update [`e2e-tests/operator_helper.go`](packages/kn-plugin-workflow/e2e-tests/operator_helper.go) so that
`InstallOperator()` and `UninstallOperator()` branch on whether `CatalogIndexImage` is set:

- **Custom mode:** create the custom CatalogSource `osl-catalog`, wait for it to be ready, then subscribe
  to `logic-operator` from it.
- **Default mode:** existing path unchanged.
- **Uninstall custom mode:** standard uninstall steps (using `logic-operator` name), then also delete `osl-catalog`.

**Expected Outcomes:**

- When `CATALOG_INDEX_IMAGE` is set before running E2E tests, the operator is installed from that index image
  as `logic-operator` with channel `stable`.
- When not set, tests behave exactly as today.
- `OPERATOR_BUNDLE_IMAGE` and `OPERATOR_IMAGE` values are printed to stdout during install for CI traceability.
- Uninstall cleans up `osl-catalog` when custom mode was used.

**Todo List:**

1. In `installOperator()` in [`e2e-tests/operator_helper.go`](packages/kn-plugin-workflow/e2e-tests/operator_helper.go),
   check if `CatalogIndexImage != ""`.
2. If yes (custom/product mode):
   a. Print the three image values for traceability (mark any that are `"not set"`).
   b. Call `operatorManager.CreateCustomCatalogSource("osl-catalog", "olm", CatalogIndexImage)`.
   c. Call `waitForCatalogSourceReady("osl-catalog", "olm")` (new helper — see step 5).
   d. Call `operatorManager.InstallOperatorFromCatalog("logic-operator", "stable", "osl-catalog", "olm")`.
3. If no (default mode): call `operator.NewInstallOperatorCommand().Execute()` as today.
4. In `uninstallOperator()`, after the standard uninstall steps, if `CatalogIndexImage != ""`, call
   `operatorManager.RemoveCustomCatalogSource("osl-catalog", "olm")`.
5. Add `waitForCatalogSourceReady(name, namespace string)` helper that polls
   `status.connectionState.lastObservedState` on the `CatalogSource` resource until it equals `"READY"`
   or the 5-minute timeout fires — mirror the pattern of the existing `waitForOperatorReady`.
6. The `operatorManager` package-level variable (used for `ListOperatorResources`, `RemoveSubscription`, etc.)
   in `waitForOperatorReady` and `uninstallOperator` already uses `common.NewOperatorManager("")` — this
   can call the new methods directly.

**Relevant Context:**

- [`e2e-tests/operator_helper.go`](packages/kn-plugin-workflow/e2e-tests/operator_helper.go) — all functions.
- [`e2e-tests/test_env.go`](packages/kn-plugin-workflow/e2e-tests/test_env.go) — `CatalogIndexImage` etc. (Sub-Task 2).
- `waitForOperatorReady` in the same file — pattern to follow for the CatalogSource readiness poll.
- Sub-Tasks 3, 4, 5 must be complete before this sub-task is implemented.
