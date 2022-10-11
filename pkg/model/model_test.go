/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sustainable-computing-io/kepler/pkg/power/rapl/source"
)
var _ = Describe("Test Model Unit", func() {
	It("Get pod power with no dependency and no node power ", func() {
		// collector/metrics.go - getEstimatorMetrics
		InitMetricIndexes(usageMetrics)
		InitEstimateFunctions(usageMetrics, systemFeatures, systemValues)
		Expect(PodComponentPowerModelValid).To(Equal(true))
		// collector/reader.go
		totalNodePower := uint64(0)
		totalGPUPower := uint64(0)
		nodeComponentPowers := source.RAPLPower{}
		podComponentPowers, podOtherPowers := GetPodPowers(usageValues, systemValues, totalNodePower, totalGPUPower, nodeComponentPowers)
		Expect(len(podOtherPowers)).To(Equal(len(usageValues)))
		Expect(len(podComponentPowers)).Should(Equal(len(usageValues)))
		fmt.Printf("Trained: %v, %v\n", podComponentPowers, podOtherPowers)
	})
	It("Get pod power with no dependency but with total node power ", func() {
		// collector/metrics.go - getEstimatorMetrics
		InitMetricIndexes(usageMetrics)
		InitEstimateFunctions(usageMetrics, systemFeatures, systemValues)
		Expect(PodComponentPowerModelValid).To(Equal(true))
		// collector/reader.go
		totalNodePower := uint64(10000)
		totalGPUPower := uint64(1000)
		nodeComponentPowers := source.RAPLPower{}
		podComponentPowers, podOtherPowers := GetPodPowers(usageValues, systemValues, totalNodePower, totalGPUPower, nodeComponentPowers)
		Expect(len(podOtherPowers)).To(Equal(len(usageValues)))
		Expect(len(podComponentPowers)).Should(Equal(len(usageValues)))
		fmt.Printf("Ratio with only total node power: %v, %v\n", podComponentPowers, podOtherPowers)
	})
	It("Get pod power with no dependency but with all node power ", func() {
		// collector/metrics.go - getEstimatorMetrics
		InitMetricIndexes(usageMetrics)
		InitEstimateFunctions(usageMetrics, systemFeatures, systemValues)
		Expect(PodComponentPowerModelValid).To(Equal(true))
		// collector/reader.go
		totalNodePower := uint64(10000)
		totalGPUPower := uint64(1000)
		nodeComponentPowers := source.RAPLPower{
			Pkg: 8000,
			Core: 5000,
			DRAM: 1000,
		}
		podComponentPowers, podOtherPowers := GetPodPowers(usageValues, systemValues, totalNodePower, totalGPUPower, nodeComponentPowers)
		Expect(len(podOtherPowers)).To(Equal(len(usageValues)))
		Expect(len(podComponentPowers)).Should(Equal(len(usageValues)))
		fmt.Printf("Ratio: %v, %v\n", podComponentPowers, podOtherPowers)
	})
})