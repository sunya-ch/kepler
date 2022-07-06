package model

import (
    "time"
	"encoding/json"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"log"
	"os"
)

const (
	POD_MEASUREMENT = "pod_energy_stat"
	ALL_PKG = "all"

	DB_ENDPOINT_KEY = "INFLUXDB_ENDPOINT"
	DB_TOKEN_KEY = "INFLUXDB_TOKEN"
	DB_ORG_KEY = "INFLUXDB_ORG"
	DB_BUCKET_KEY = "INFLUXDB_BUCKET"

	DEFAULT_ENDPOINT = ""
	DEFAULT_TOKEN = ""
	DEFAULT_ORG = "influxdata"
	DEFAULT_BUCKET = "default"
)

var InfluxDBEndpoint, InfluxDBToken, InfluxDBOrg, InfluxDBBucket string

type PodPoint struct {
	Tag 	PodTag
	Fields 	map[string]interface{}
}

type PodTag struct {
	Name 		string 	`json:"pod_name"`
	Namespace 	string 	`json:"pod_namespace"`
	Command 	string 	`json:"command"`
	CorePower   string `json:"core_power"`
	DRAMPower   string `json:"dram_power"`
	GPUPower    string `json:"gpu_power"`
	OtherPower  string `json:"other_power"`
	PKGS        string 	`json:"pkg_str"`
}

type InfluxDBClient struct {
	influxdb2.Client
	api.WriteAPI
}

func setFromEnv(key, defaultValue string) string {
	setValue, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return setValue
}

func SetInfluxServerEndpoint(){
	InfluxDBEndpoint = setFromEnv(DB_ENDPOINT_KEY, DEFAULT_ENDPOINT)
	InfluxDBToken = setFromEnv(DB_TOKEN_KEY, DEFAULT_TOKEN)
	InfluxDBOrg = setFromEnv(DB_ORG_KEY, DEFAULT_ORG)
	InfluxDBBucket = setFromEnv(DB_BUCKET_KEY, DEFAULT_BUCKET)
}

func NewInfluxDBClient() *InfluxDBClient {
	client := influxdb2.NewClient(InfluxDBEndpoint, InfluxDBToken)
	writeAPI := client.WriteAPI(InfluxDBOrg, InfluxDBBucket)
	if writeAPI == nil {
		log.Println("cannot initialize InfluxDBClient.")
		client.Close()
		return nil
	}
	log.Println("successfully initialize InfluxDBClient.")
	return &InfluxDBClient {
		Client: client,
		WriteAPI: writeAPI,
	}
}

func podTagToMap(tag PodTag) map[string]string {
	tagByte, _ := json.Marshal(tag)
	var tagMap map[string]string
	json.Unmarshal(tagByte, &tagMap)
	return tagMap
}

func (c *InfluxDBClient) Write(points []PodPoint) {
	for _, point := range points {
		strMap := podTagToMap(point.Tag)
		p := influxdb2.NewPoint(POD_MEASUREMENT, strMap, point.Fields, time.Now())
		c.WriteAPI.WritePoint(p)
	}
	c.WriteAPI.Flush()
}

func (c *InfluxDBClient) Destroy() {
	c.Client.Close()
}