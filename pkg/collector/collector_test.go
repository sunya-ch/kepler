package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
)

const (
	CPU_USAGE_TOTAL_KEY = "cgroupfs_cpu_usage_us"
	SAMPLE_CURR         = 100
	SAMPLE_AGGR         = 1000
	SAMPLE_NODE_ENERGY  = 20000
	SAMPLE_FREQ         = 100000
)

func convertPromMetricToMap(body []byte, metric string) map[string]string {
	regStr := fmt.Sprintf(`%s{[^{}]*}`, metric)
	r := regexp.MustCompile(regStr)
	match := r.FindString(string(body))
	match = strings.Replace(match, metric, "", 1)
	match = strings.ReplaceAll(match, "=", `"=`)
	match = strings.ReplaceAll(match, ",", `,"`)
	match = strings.ReplaceAll(match, "{", `{"`)
	match = strings.ReplaceAll(match, "=", `:`)
	var response map[string]string
	json.Unmarshal([]byte(match), &response)
	return response
}

func convertPromToValue(body []byte, metric string) (int, error) {
	regStr := fmt.Sprintf(`%s{[^{}]*} [0-9]+`, metric)
	r := regexp.MustCompile(regStr)
	match := r.FindString(string(body))
	splits := strings.Split(match, " ")
	fmt.Println(splits, regStr)
	return strconv.Atoi(splits[1])
}

var _ = Describe("Test Collector Unit", func() {
	It("Init and Run", func() {
		newCollector, err := New()
		Expect(err).NotTo(HaveOccurred())
		err = prometheus.Register(newCollector)
		Expect(err).NotTo(HaveOccurred())
		req, _ := http.NewRequest("GET", "", nil)
		res := httptest.NewRecorder()
		handler := http.Handler(promhttp.Handler())
		handler.ServeHTTP(res, req)
		body, _ := ioutil.ReadAll(res.Body)
		Expect(len(body)).Should(BeNumerically(">", 0))

		regStr := fmt.Sprintf(`%s{[^{}]*}`, POD_ENERGY_STAT_METRIC)
		r := regexp.MustCompile(regStr)
		match := r.FindString(string(body))
		Expect(match).To(Equal(""))

		podEnergy = map[string]*PodEnergy{
			"abcd": &PodEnergy{
				PodName:         "podA",
				Namespace:       "default",
				AggEnergyInCore: 10,
				CgroupFSStats: map[string]*UInt64Stat{
					CPU_USAGE_TOTAL_KEY: &UInt64Stat{
						Curr: SAMPLE_CURR,
						Aggr: SAMPLE_AGGR,
					},
				},
			},
		}
		nodeEnergy = map[string]float64{
			"sensor0": SAMPLE_NODE_ENERGY,
		}
		cpuFrequency = map[int32]uint64{
			0: SAMPLE_FREQ,
		}

		res = httptest.NewRecorder()
		handler = http.Handler(promhttp.Handler())
		handler.ServeHTTP(res, req)
		body, _ = ioutil.ReadAll(res.Body)
		Expect(len(body)).Should(BeNumerically(">", 0))
		fmt.Printf("Result:\n %s\n", body)

		// check sample pod energy stat
		response := convertPromMetricToMap(body, POD_ENERGY_STAT_METRIC)
		currSample, found := response["curr_"+CPU_USAGE_TOTAL_KEY]
		Expect(found).To(Equal(true))
		Expect(currSample).To(Equal(fmt.Sprintf("%d", SAMPLE_CURR)))
		aggrSample, found := response["total_"+CPU_USAGE_TOTAL_KEY]
		Expect(found).To(Equal(true))
		Expect(aggrSample).To(Equal(fmt.Sprintf("%d", SAMPLE_AGGR)))
		// check sample node energy
		val, err := convertPromToValue(body, NODE_ENERGY_METRIC)
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal(int(SAMPLE_NODE_ENERGY / 1000)))
		// check sample frequency
		val, err = convertPromToValue(body, FREQ_METRIC)
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal(int(SAMPLE_FREQ)))
	})
})
