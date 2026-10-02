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
	"fmt"
	"os"
	"time"

	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/command/operator"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/common"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/common/k8sclient"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

var operatorManager = common.NewOperatorManager("")

func orNotSet(v string) string {
	if v == "" {
		return "not set"
	}
	return v
}

func InstallOperator() {
	installOperator()
	waitForOperatorReady()
	checkOperatorInstalled()
}

func UninstallOperator() {
	uninstallOperator()
}

func installOperator() {
	if CatalogIndexImage != "" {
		// Product/OSL build: install from the custom CatalogSource.
		fmt.Println("🚀 Installing operator from custom CatalogSource (product build)...")
		fmt.Printf("   CATALOG_INDEX_IMAGE:   %s\n", CatalogIndexImage)
		fmt.Printf("   OPERATOR_BUNDLE_IMAGE: %s\n", orNotSet(OperatorBundleImage))
		fmt.Printf("   OPERATOR_IMAGE:        %s\n", orNotSet(OperatorImage))

		if err := operatorManager.CreateCustomCatalogSource(customCatalogName, customCatalogNamespace, CatalogIndexImage); err != nil {
			fmt.Println("Failed to create custom CatalogSource:", err)
			os.Exit(1)
		}

		waitForCatalogSourceReady(customCatalogName, customCatalogNamespace)

		if err := createPinnedSubscription(metadata.LogicOperatorName, "stable", customCatalogName, customCatalogNamespace, OperatorStartingCSV); err != nil {
			fmt.Println("Failed to create subscription from custom catalog:", err)
			os.Exit(1)
		}

		if OperatorStartingCSV != "" {
			approveInstallPlan(metadata.LogicOperatorName, OperatorStartingCSV)
		}
		return
	}

	// Default path: install from the public catalog via the standard CLI command.
	var install = operator.NewInstallOperatorCommand()
	err := install.Execute()
	if err != nil {
		fmt.Println("Failed to install operator:", err)
		os.Exit(1)
	}
}

func waitForCatalogSourceReady(name, namespace string) {
	dynamicClient, err := k8sclient.DynamicClient()
	if err != nil {
		fmt.Println("Failed to create dynamic client:", err)
		os.Exit(1)
	}

	ready := make(chan bool)
	defer close(ready)
	timeoutCh := time.After(5 * time.Minute)

	go func() {
		for {
			select {
			case <-timeoutCh:
				fmt.Printf("Timeout waiting for CatalogSource %q to be ready\n", name)
				os.Exit(1)
			default:
				cs, err := dynamicClient.Resource(catalogSourcesGVR).Namespace(namespace).Get(
					context.Background(), name, v1.GetOptions{})
				if err != nil {
					time.Sleep(5 * time.Second)
					continue
				}

				state, _, _ := unstructured.NestedString(cs.Object,
					"status", "connectionState", "lastObservedState")
				if state == "READY" {
					ready <- true
					return
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()

	select {
	case <-ready:
		fmt.Printf(" - ✅ CatalogSource %q is ready\n", name)
	}
}

func waitForOperatorReady() {
	deployed := make(chan bool)
	defer close(deployed)
	timeoutCh := time.After(5 * time.Minute)

	go func() {
		for {
			select {
			case <-timeoutCh:
				fmt.Println("Timeout waiting for operator to be ready")
				os.Exit(1)
			default:
				resources, err := operatorManager.ListOperatorResources()
				if err != nil {
					fmt.Println("Failed to list operator resources:", err)
					os.Exit(1)
				}

				if len(resources) == 0 {
					continue
				}

				var ready = true
				for _, resource := range resources {
					phase, found, err := unstructured.NestedString(resource.Object, "status", "phase")
					if !found {
						ready = false
					}
					if err != nil {
						fmt.Println("Failed to get resource status:", err)
						os.Exit(1)
					}
					if phase != "Succeeded" {
						ready = false
					}
				}

				if ready {
					deployed <- true
					return
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()

	select {
	case <-deployed:
		fmt.Printf(" - ✅ Operator is ready\n")
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
func approveInstallPlan(subscriptionName, csvName string) {
	dynamicClient, err := k8sclient.DynamicClient()
	if err != nil {
		fmt.Println("Failed to create dynamic client:", err)
		os.Exit(1)
	}

	fmt.Printf("⏳ Waiting for InstallPlan for CSV %q to appear...\n", csvName)
	timeoutCh := time.After(5 * time.Minute)

	for {
		select {
		case <-timeoutCh:
			fmt.Printf("Timeout waiting for InstallPlan for CSV %q\n", csvName)
			os.Exit(1)
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
						fmt.Printf("Failed to approve InstallPlan %q: %v\n", planName, err)
						os.Exit(1)
					}
					fmt.Printf(" - ✅ InstallPlan %q approved\n", planName)
					return
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}
