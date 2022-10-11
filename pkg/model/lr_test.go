package model

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

var (
	SampleCategoricalFeatures = map[string]CategoricalFeature{
		"Sandy Bridge": {
			Weight: 1.0,
		},
	}
	SampleCoreNumericalVars = map[string]NormalizedNumericalFeature{
		"cpu_cycles": {Weight: 1.0, Mean: 0, Variance: 1},
	}
	SampleDramNumbericalVars = map[string]NormalizedNumericalFeature{
		"cache_miss": {Weight: 1.0, Mean: 0, Variance: 1},
	}
	SampleComponentWeightResponse = ComponentModelWeights{
		"core": genWeights(SampleCoreNumericalVars),
		"dram": genWeights(SampleDramNumbericalVars),
	}
	SamplePowerWeightResponse = genWeights(SampleCoreNumericalVars)

	modelServerPort = 8100
)

func genWeights(numericalVars map[string]NormalizedNumericalFeature) ModelWeights {
	return ModelWeights{
		AllWeights{
			BiasWeight:           1.0,
			CategoricalVariables: map[string]map[string]CategoricalFeature{"cpu_architecture": SampleCategoricalFeatures},
			NumericalVariables:   numericalVars,
		},
	}
}

func getDummyWeights(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var req ModelRequest
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		panic(err)
	}
	if strings.Contains(req.OutputType, "ComponentModelWeight") {
		err = json.NewEncoder(w).Encode(SampleComponentWeightResponse)
	} else {
		err = json.NewEncoder(w).Encode(SamplePowerWeightResponse)
	}
	if err != nil {
		panic(err)
	}
}

func getHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
}

func dummyModelWeightServer(start, quit chan bool) {
	http.HandleFunc("/model", getDummyWeights)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", modelServerPort))
	if err != nil {
		fmt.Println(modelServerPort)
		panic(err)
	}
	defer listener.Close()
	go func() {
		<-quit
		listener.Close()
	}()
	start <- true
	log.Printf("Server ends: %v\n", http.Serve(listener, getHandler(http.DefaultServeMux)))
}

func genLinearRegressor(outputType ModelOutputType, endpoint, initModelURL string) LinearRegressor {
	return LinearRegressor{
		Endpoint:       endpoint,
		UsageMetrics:   usageMetrics,
		OutputType:     outputType,
		SystemFeatures: systemFeatures,
		InitModelURL:   initModelURL,
	}
}

var _ = Describe("Test LR Weight Unit", func() {
	It("UseWeightFromModelServer", func() {
		start := make(chan bool)
		quit := make(chan bool)
		defer close(quit)
		defer close(start)
		go dummyModelWeightServer(start, quit)
		<-start

		// NodeTotalPower
		endpoint := "http://127.0.0.1:8100/model"
		r := genLinearRegressor(AbsModelWeight, endpoint, "")
		valid := r.Init()
		Expect(valid).To(Equal(true))
		powers, err := r.GetTotalPower([][]float64{nodeUsageValue}, systemValues)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(powers)).Should(Equal(1))
		Expect(powers[0]).Should(BeEquivalentTo(3))

		// NodeComponentPower
		r = genLinearRegressor(AbsComponentModelWeight, endpoint, "")
		valid = r.Init()
		Expect(valid).To(Equal(true))
		compPowers, err := r.GetComponentPower([][]float64{nodeUsageValue}, systemValues)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(compPowers["core"])).Should(Equal(1))
		Expect(compPowers["core"][0]).Should(BeEquivalentTo(3))

		// PodTotalPower
		r = genLinearRegressor(DynModelWeight, endpoint, "")
		valid = r.Init()
		Expect(valid).To(Equal(true))
		powers, err = r.GetTotalPower(usageValues, systemValues)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(powers)).Should(Equal(len(usageValues)))
		Expect(powers[0]).Should(BeEquivalentTo(3))

		// PodComponentPower
		r = genLinearRegressor(DynComponentModelWeight, endpoint, "")
		valid = r.Init()
		Expect(valid).To(Equal(true))
		compPowers, err = r.GetComponentPower(usageValues, systemValues)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(compPowers["core"])).Should(Equal(len(usageValues)))
		Expect(compPowers["core"][0]).Should(BeEquivalentTo(3))

		quit <- true
	})
	It("UseInitModelURL", func() {
		// NodeComponentPower
		initModelURL := "https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/AbsComponentModelWeight/Full/KerasCompWeightFullPipeline/KerasCompWeightFullPipeline.json"
		r := genLinearRegressor(AbsComponentModelWeight, "", initModelURL)
		valid := r.Init()
		Expect(valid).To(Equal(true))
		_, err := r.GetComponentPower([][]float64{nodeUsageValue}, systemValues)
		Expect(err).NotTo(HaveOccurred())

		// PodComponentPower
		initModelURL = "https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/DynComponentModelWeight/CgroupOnly/ScikitMixed/ScikitMixed.json"
		r = genLinearRegressor(DynComponentModelWeight, "", initModelURL)
		valid = r.Init()
		Expect(valid).To(Equal(true))
		_, err = r.GetComponentPower(usageValues, systemValues)
		Expect(err).NotTo(HaveOccurred())
	})
})
