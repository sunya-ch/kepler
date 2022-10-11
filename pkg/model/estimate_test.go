package model

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"net"
	"os"
	"strings"
	"fmt"
)

var (
	SampleDynPowerValue float64 = 100.0
)

func dummyEstimator(serveSocket string, start, quit chan bool) {
	cleanup := func() {
		if _, err := os.Stat(serveSocket); err == nil {
			if err := os.RemoveAll(serveSocket); err != nil {
				panic(err)
			}
		}
	}
	cleanup()

	listener, err := net.Listen("unix", serveSocket)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	go func() {
		<-quit
		cleanup()
	}()

	start <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Close dummy estimator %v\n", err)
			break
		}
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
		var powerRequest PowerRequest
		err = json.Unmarshal(buf[0:n], &powerRequest)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%v\n", powerRequest)
		powers := make([]float64, len(powerRequest.UsageValues))
		powers[0] = SampleDynPowerValue
		msg := ""
		var powerResponseJSON []byte
		if strings.Contains(powerRequest.OutputType, "Component") {
			powerResponse := ComponentPowerResponse{
				Powers:  map[string][]float64{"pkg": powers},
				Message: msg,
			}
			powerResponseJSON, err = json.Marshal(powerResponse)
		} else {
			powerResponse := TotalPowerResponse{
				Powers:  powers,
				Message: msg,
			}
			powerResponseJSON, err = json.Marshal(powerResponse)
		}
		if err != nil {
			panic(err)
		}
		conn.Write(powerResponseJSON)
		conn.Close()
	}
}

func genEstimatorSidecarConnector(serveSocket string, outputType ModelOutputType) EstimatorSidecarConnector {
	return EstimatorSidecarConnector{
		Socket:         serveSocket,
		UsageMetrics:   usageMetrics,
		OutputType:     outputType,
		SystemFeatures: systemFeatures,
	}
}

var _ = Describe("Test Estimate Unit", func() {
	It("GetNodeTotalPowerByEstimator", func() {
		serveSocket := "/tmp/node-total-power.sock"
		start := make(chan bool)
		quit := make(chan bool)
		defer close(quit)
		defer close(start)
		go dummyEstimator(serveSocket, start, quit)
		<-start

		c := genEstimatorSidecarConnector(serveSocket, AbsPower)
		valid := c.Init(systemValues)
		Expect(valid).To(Equal(true))
		powers, err := c.GetTotalPower([][]float64{nodeUsageValue}, systemValues)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(powers)).Should(Equal(1))
		Expect(powers[0]).Should(Equal(SampleDynPowerValue))
		quit <- true
	})
	It("GetPodTotalPowerByEstimator", func() {
		serveSocket := "/tmp/pod-total-power.sock"
		start := make(chan bool)
		quit := make(chan bool)
		defer close(quit)
		defer close(start)
		go dummyEstimator(serveSocket, start, quit)
		<-start

		c := genEstimatorSidecarConnector(serveSocket, DynPower)
		valid := c.Init(systemValues)
		Expect(valid).To(Equal(true))
		powers, err := c.GetTotalPower(usageValues, systemValues)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(powers)).Should(Equal(len(usageValues)))
		Expect(powers[0]).Should(Equal(SampleDynPowerValue))
		quit <- true
	})
	It("GetNodeComponentPowerByEstimator", func() {
		serveSocket := "/tmp/node-comp-power.sock"
		start := make(chan bool)
		quit := make(chan bool)
		defer close(quit)
		defer close(start)
		go dummyEstimator(serveSocket, start, quit)
		<-start

		c := genEstimatorSidecarConnector(serveSocket, AbsComponentPower)
		valid := c.Init(systemValues)
		Expect(valid).To(Equal(true))
		powers, err := c.GetComponentPower([][]float64{nodeUsageValue}, systemValues)
		Expect(err).NotTo(HaveOccurred())
		pkgPowers, ok := powers["pkg"]
		Expect(ok).To(Equal(true))
		Expect(len(pkgPowers)).Should(Equal(1))
		Expect(pkgPowers[0]).Should(Equal(SampleDynPowerValue))
		quit <- true
	})
	It("GetPodComponentPowerByEstimator", func() {
		serveSocket := "/tmp/pod-comp-power.sock"
		start := make(chan bool)
		quit := make(chan bool)
		defer close(quit)
		defer close(start)
		go dummyEstimator(serveSocket, start, quit)
		<-start

		c := genEstimatorSidecarConnector(serveSocket, DynComponentPower)
		valid := c.Init(systemValues)
		Expect(valid).To(Equal(true))
		powers, err := c.GetComponentPower(usageValues, systemValues)
		Expect(err).NotTo(HaveOccurred())
		pkgPowers, ok := powers["pkg"]
		Expect(ok).To(Equal(true))
		Expect(len(pkgPowers)).Should(Equal(len(usageValues)))
		Expect(pkgPowers[0]).Should(Equal(SampleDynPowerValue))
		quit <- true
	})
})
