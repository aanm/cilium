// Copyright 2017-2019 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sTest

import (
	"fmt"

	. "github.com/cilium/cilium/test/ginkgo-ext"
	"github.com/cilium/cilium/test/helpers"
)

var upgradeTest = func() {
	var (
		kubectl *helpers.Kubectl

		cleanupCallback = func() {}
	)

	BeforeAll(func() {
		kubectl = helpers.CreateKubectl(helpers.K8s1VMName(), logger)

		demoPath = helpers.ManifestGet(kubectl.BasePath(), "demo.yaml")
		l7Policy = helpers.ManifestGet(kubectl.BasePath(), "l7-policy.yaml")
		migrateSVCClient = helpers.ManifestGet(kubectl.BasePath(), "migrate-svc-client.yaml")
		migrateSVCServer = helpers.ManifestGet(kubectl.BasePath(), "migrate-svc-server.yaml")

		kubectl.Delete(migrateSVCClient)
		kubectl.Delete(migrateSVCServer)
		kubectl.Delete(l7Policy)
		kubectl.Delete(demoPath)

		// Delete kube-dns because if not will be a restore the old endpoints
		// from master instead of create the new ones.
		_ = kubectl.DeleteResource(
			"deploy", fmt.Sprintf("-n %s kube-dns", helpers.KubeSystemNamespace))

		_ = kubectl.DeleteResource(
			"deploy", fmt.Sprintf("-n %s cilium-operator", helpers.CiliumNamespace))
		// Sometimes PolicyGen has a lot of pods running around without delete
		// it. Using this we are sure that we delete before this test start
		kubectl.Exec(fmt.Sprintf(
			"%s delete --all pods,svc,cnp -n %s", helpers.KubectlCmd, helpers.DefaultNamespace))

		ExpectAllPodsTerminated(kubectl)
	})

	AfterAll(func() {
		kubectl.CloseSSHClient()
	})

	AfterFailed(func() {
		kubectl.CiliumReport("cilium endpoint list")
	})

	JustAfterEach(func() {
		kubectl.ValidateNoErrorsInLogs(CurrentGinkgoTestDescription().Duration)
	})

	AfterEach(func() {
		cleanupCallback()
		ExpectAllPodsTerminated(kubectl)
	})

	for imageVersion, chartVersion := range helpers.NightlyStableUpgradesFrom {
		func(imageVersion, chartVersion string) {
			SkipItIf(func() bool { return !helpers.RunsWithKubeProxy() },
				fmt.Sprintf("Update Cilium from %s to master", imageVersion), func() {
					var assertUpgradeSuccessful func()
					assertUpgradeSuccessful, cleanupCallback = InstallAndValidateCiliumUpgrades(
						kubectl,
						chartVersion,
						imageVersion,
						helpers.CiliumLatestHelmChartVersion,
						helpers.GetLatestImageVersion(),
					)
					assertUpgradeSuccessful()
				})
		}(imageVersion, chartVersion)
	}
}
