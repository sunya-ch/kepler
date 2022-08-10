package collector

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Collector Unit", func() {
	It("Check feature values", func() {
		Expect(len(uintFeatures)).Should(BeNumerically(">", 0))
		Expect(len(collectedLabel)).Should(BeNumerically(">", 0))
		Expect(len(podEnergyLabels)).Should(BeNumerically(">", 0))
		fmt.Printf("%v\n%v\n%v\n", uintFeatures, collectedLabel, podEnergyLabels)
	})

	It("Check convert values", func() {

		podEnergy := &PodEnergy{
			PodName:         "podA",
			Namespace:       "default",
			AggEnergyInCore: 10,
			CgroupFSStats: map[string]*UInt64Stat{
				CPU_USAGE_TOTAL_KEY: &UInt64Stat{
					Curr: SAMPLE_CURR,
					Aggr: SAMPLE_AGGR,
				},
			}}

		collectedValues := convertCollectedValues(FLOAT_FEATURES, uintFeatures, podEnergy)
		Expect(len(collectedValues)).To(Equal(len(collectedLabel)))
		fmt.Printf("%v\n", collectedValues)
	})

})
