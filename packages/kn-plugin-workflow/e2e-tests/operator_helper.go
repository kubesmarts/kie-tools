//go:build e2e_tests

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package e2e_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/command/operator"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/common"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/common/k8sclient"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/metadata"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	customCatalogName      = "osl-catalog"
	customCatalogNamespace = "olm"
)

var catalogSourcesGVR = schema.GroupVersionResource{
	Group:    "operators.coreos.com",
	Version:  "v1alpha1",
	Resource: "catalogsources",
}

var installPlansGVR = schema.GroupVersionResource{
	Group:    "operators.coreos.com",
	Version:  "v1alpha1",
	Resource: "installplans",
}

var subscriptionsGVR = schema.GroupVersionResource{
	Group:    "operators.coreos.com",
	Version:  "v1alpha1",
	Resource: "subscriptions",
}

var clusterServiceVersionsGVR = schema.GroupVersionResource{
	Group:    "operators.coreos.com",
	Version:  "v1alpha1",
	Resource: "clusterserviceversions",
}

var operatorManager = common.NewOperatorManager("")

func orNotSet(v string) string {
	if v == "" {
		return "not set"
	}
	return v
}

// operatorLogsDir is the directory where operator diagnostic snapshots are written.
// go test runs with cwd = the package source dir (e2e-tests/), so we go up one level
// to reach the package root and then into dist-tests-e2e — keeping all test artefacts
// together for CI collection.
const operatorLogsDir = "../../dist-tests-e2e/operator-logs"

func InstallOperator() {
	if err := installOperator(); err != nil {
		fmt.Println("❌ Operator installation failed:", err)
		collectOperatorLogs("install-failure")
		os.Exit(1)
	}
	if err := waitForOperatorReady(); err != nil {
		fmt.Println("❌ Operator did not become ready:", err)
		collectOperatorLogs("install-failure")
		os.Exit(1)
	}
	checkOperatorInstalled()
	collectOperatorLogs("post-install")
}

func UninstallOperator() {
	collectOperatorLogs("pre-uninstall")
	uninstallOperator()
}

func installOperator() error {
	if CatalogIndexImage != "" {
		// Product/OSL build: install from the custom CatalogSource.
		fmt.Println("🚀 Installing operator from custom CatalogSource (product build)...")
		fmt.Printf("   CATALOG_INDEX_IMAGE:   %s\n", CatalogIndexImage)
		fmt.Printf("   OPERATOR_BUNDLE_IMAGE: %s\n", orNotSet(OperatorBundleImage))
		fmt.Printf("   OPERATOR_IMAGE:        %s\n", orNotSet(OperatorImage))

		if err := operatorManager.CreateCustomCatalogSource(customCatalogName, customCatalogNamespace, CatalogIndexImage); err != nil {
			return fmt.Errorf("failed to create custom CatalogSource: %w", err)
		}

		if err := waitForCatalogSourceReady(customCatalogName, customCatalogNamespace); err != nil {
			return fmt.Errorf("CatalogSource not ready: %w", err)
		}

		if err := createPinnedSubscription(metadata.LogicOperatorName, "stable", customCatalogName, customCatalogNamespace, OperatorStartingCSV); err != nil {
			return fmt.Errorf("failed to create subscription from custom catalog: %w", err)
		}

		if OperatorStartingCSV != "" {
			if err := approveInstallPlan(metadata.LogicOperatorName, OperatorStartingCSV); err != nil {
				return fmt.Errorf("failed to approve InstallPlan: %w", err)
			}
		}
		return nil
	}

	// Default path: install from the public catalog via the standard CLI command.
	var install = operator.NewInstallOperatorCommand()
	if err := install.Execute(); err != nil {
		return fmt.Errorf("failed to install operator: %w", err)
	}
	return nil
}

func waitForCatalogSourceReady(name, namespace string) error {
	dynamicClient, err := k8sclient.DynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	timeoutCh := time.After(5 * time.Minute)
	for {
		select {
		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for CatalogSource %q to be ready", name)
		default:
		}

		cs, err := dynamicClient.Resource(catalogSourcesGVR).Namespace(namespace).Get(
			context.Background(), name, v1.GetOptions{})
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		state, _, _ := unstructured.NestedString(cs.Object,
			"status", "connectionState", "lastObservedState")
		if state == "READY" {
			fmt.Printf(" - ✅ CatalogSource %q is ready\n", name)
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}

func waitForOperatorReady() error {
	timeoutCh := time.After(5 * time.Minute)
	for {
		select {
		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for operator to become ready")
		default:
		}

		resources, err := operatorManager.ListOperatorResources()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if len(resources) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		ready := true
		for _, resource := range resources {
			phase, _, err := unstructured.NestedString(resource.Object, "status", "phase")
			if err != nil {
				return fmt.Errorf("failed to get resource status: %w", err)
			}
			if phase != "Succeeded" {
				ready = false
			}
		}

		if ready {
			fmt.Printf(" - ✅ Operator is ready\n")
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}

func checkOperatorInstalled() {
	if CatalogIndexImage != "" {
		// In custom-catalog (product) mode the Subscription is named "logic-operator",
		// not "sonataflow-operator", so the standard status command would fail.
		// Operator readiness was already confirmed by waitForOperatorReady().
		fmt.Println(" - ✅ Operator status check skipped in custom catalog mode (readiness already confirmed)")
		return
	}
	var status = operator.NewStatusOperatorCommand()
	err := status.Execute()
	if err != nil {
		fmt.Println("Failed to check operator status:", err)
		os.Exit(1)
	}
}

func uninstallOperator() {
	if CatalogIndexImage != "" {
		// In custom-catalog (product) mode use the logic-operator name directly.
		fmt.Println("🚀 Uninstalling the logic-operator (product build)...")
		// Remove CRDs first — reads the subscription to discover owned CRDs via the installed CSV.
		if err := operatorManager.RemoveCRDBySubscription(metadata.LogicOperatorName); err != nil {
			fmt.Println("Failed to remove CRDs:", err)
			os.Exit(1)
		}
		if err := operatorManager.RemoveCSVByPrefix(metadata.LogicOperatorName); err != nil {
			fmt.Println("Failed to remove CSV:", err)
			os.Exit(1)
		}
		if err := operatorManager.RemoveSubscriptionByName(metadata.LogicOperatorName); err != nil {
			fmt.Println("Failed to remove subscription:", err)
			os.Exit(1)
		}
		if err := operatorManager.RemoveCustomCatalogSource(customCatalogName, customCatalogNamespace); err != nil {
			fmt.Println("Failed to remove custom CatalogSource:", err)
			os.Exit(1)
		}
		fmt.Println("🎉 logic-operator successfully uninstalled.")
		return
	}

	var uninstall = operator.NewUnInstallOperatorCommand()
	err := uninstall.Execute()
	if err != nil {
		fmt.Println("Failed to uninstall operator:", err)
		os.Exit(1)
	}
}

// createPinnedSubscription creates an OLM Subscription for the given operator. When startingCSV
// is non-empty it pins the subscription to that exact CSV and sets installPlanApproval to Manual
// so OLM never auto-upgrades to a newer version during the test run.
// This is intentionally kept in the test helper (not in pkg/common) because pinning is a
// test-only concern: production installs should always follow the channel head.
func createPinnedSubscription(operatorName, channel, source, sourceNamespace, startingCSV string) error {
	dynamicClient, err := k8sclient.DynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	spec := map[string]interface{}{
		"channel":         channel,
		"name":            operatorName,
		"source":          source,
		"sourceNamespace": sourceNamespace,
	}
	if startingCSV != "" {
		spec["startingCSV"] = startingCSV
		spec["installPlanApproval"] = "Manual"
		fmt.Printf("🔧 Pinning subscription to CSV %q (installPlanApproval: Manual)\n", startingCSV)
	}

	sub := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]interface{}{
				"name":      operatorName,
				"namespace": "operators",
			},
			"spec": spec,
		},
	}

	_, err = dynamicClient.Resource(subscriptionsGVR).Namespace("operators").Create(
		context.Background(), sub, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create subscription %q: %v", operatorName, err)
	}
	fmt.Println("✅ Subscription created successfully")
	return nil
}

// approveInstallPlan waits for an InstallPlan referencing the given CSV to appear in the
// subscription's namespace, then patches it to approved=true so that a Manual-approval
// subscription actually installs the operator.
func approveInstallPlan(subscriptionName, csvName string) error {
	dynamicClient, err := k8sclient.DynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	fmt.Printf("⏳ Waiting for InstallPlan for CSV %q to appear...\n", csvName)
	timeoutCh := time.After(5 * time.Minute)

	for {
		select {
		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for InstallPlan for CSV %q", csvName)
		default:
		}

		plans, err := dynamicClient.Resource(installPlansGVR).Namespace("operators").List(
			context.Background(), v1.ListOptions{})
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, plan := range plans.Items {
			clusterServiceVersionNames, _, _ := unstructured.NestedStringSlice(plan.Object, "spec", "clusterServiceVersionNames")
			for _, name := range clusterServiceVersionNames {
				if name == csvName {
					planName := plan.GetName()
					fmt.Printf(" - Approving InstallPlan %q for CSV %q...\n", planName, csvName)
					plan.Object["spec"].(map[string]interface{})["approved"] = true
					_, err := dynamicClient.Resource(installPlansGVR).Namespace("operators").Update(
						context.Background(), &plan, v1.UpdateOptions{})
					if err != nil {
						return fmt.Errorf("failed to approve InstallPlan %q: %w", planName, err)
					}
					fmt.Printf(" - ✅ InstallPlan %q approved\n", planName)
					return nil
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// collectOperatorLogs gathers diagnostic information about the operator installation and writes
// it to operatorLogsDir/<phase>/. Collected artefacts:
//   - Pod logs for every pod in the "operators" namespace (the operator itself)
//   - Pod logs for OLM system pods in the "olm" namespace (catalog-operator, olm-operator)
//   - JSON snapshots of Subscription, CSV, InstallPlan, and CatalogSource objects
//
// Errors are only printed as warnings — log collection must never fail the test run.
func collectOperatorLogs(phase string) {
	dir := filepath.Join(operatorLogsDir, phase)
	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Printf("⚠️  operator-logs: cannot create directory %q: %v\n", dir, err)
		return
	}
	fmt.Printf("📋 Collecting operator diagnostic logs → %s/\n", dir)

	restConfig, err := k8sclient.KubeRestConfig()
	if err != nil {
		fmt.Printf("⚠️  operator-logs: cannot build REST config: %v\n", err)
		return
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		fmt.Printf("⚠️  operator-logs: cannot create clientset: %v\n", err)
		return
	}
	dynamicClient, err := k8sclient.DynamicClient()
	if err != nil {
		fmt.Printf("⚠️  operator-logs: cannot create dynamic client: %v\n", err)
		return
	}

	// --- Pod logs ---
	for _, ns := range []string{"operators", "olm"} {
		pods, err := clientset.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{})
		if err != nil {
			fmt.Printf("⚠️  operator-logs: cannot list pods in %q: %v\n", ns, err)
			continue
		}
		for _, pod := range pods.Items {
			for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
				logFile := filepath.Join(dir, fmt.Sprintf("%s_%s_%s.log", ns, pod.Name, container.Name))
				req := clientset.CoreV1().Pods(ns).GetLogs(pod.Name, &corev1.PodLogOptions{
					Container: container.Name,
				})
				stream, err := req.Stream(context.Background())
				if err != nil {
					// Pod may not have started yet — write a note and continue.
					_ = os.WriteFile(logFile, []byte(fmt.Sprintf("(log unavailable: %v)\n", err)), 0640)
					continue
				}
				data, _ := io.ReadAll(stream)
				stream.Close()
				if err := os.WriteFile(logFile, data, 0640); err != nil {
					fmt.Printf("⚠️  operator-logs: cannot write %q: %v\n", logFile, err)
				}
			}
		}
	}

	// --- OLM resource snapshots ---
	type gvrTarget struct {
		gvr       schema.GroupVersionResource
		namespace string
		label     string
	}
	targets := []gvrTarget{
		{subscriptionsGVR, "operators", "subscriptions"},
		{clusterServiceVersionsGVR, "operators", "clusterserviceversions"},
		{installPlansGVR, "operators", "installplans"},
		{catalogSourcesGVR, "olm", "catalogsources"},
	}
	for _, t := range targets {
		list, err := dynamicClient.Resource(t.gvr).Namespace(t.namespace).List(
			context.Background(), v1.ListOptions{})
		if err != nil {
			fmt.Printf("⚠️  operator-logs: cannot list %s: %v\n", t.label, err)
			continue
		}
		data, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			continue
		}
		outFile := filepath.Join(dir, t.label+".json")
		if err := os.WriteFile(outFile, data, 0640); err != nil {
			fmt.Printf("⚠️  operator-logs: cannot write %q: %v\n", outFile, err)
		}
	}

	fmt.Printf("📋 Operator diagnostic logs collected in %s/\n", dir)
}
