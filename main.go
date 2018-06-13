package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/abhisheknsit/boomer/boomer"
)

var (
	httpClient          *http.Client
	maxIdleConnections  int
	testDefinitionsFile string
	suiteName           []string
	postData            []byte
)

const (
	RequestTimeout int = 30
)

// init HTTPClient
func init() {
	httpClient = createHTTPClient()
}

type weightParams struct {
	magnitude int
	frequency int
	constant  int
	phase     int
}

type header struct {
	name  string
	value string
}

type Test struct {
	Url     string       `json:"url,omitempty"`
	Headers []header     `json:"headers,omitempty"`
	Body    int64        `json:"body,omitempty"`
	Weight  weightParams `json:"weight,omitempty"`
	Method  string       `json:"method,omitempty"`
}

type suite struct {
	suite []Test
}

func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: maxIdleConnections,
			MaxIdleConns:        maxIdleConnections,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}

func httpget(url string, headers []header) {

	start := boomer.Now()
	resp, err := http.Get(url)
	elapsed := boomer.Now() - start

	if err != nil {
		boomer.Events.Publish("request_failure", "get", url, elapsed, err.Error())
	} else {
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body)
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			boomer.Events.Publish("request_failure", "get", url, elapsed, strconv.Itoa(resp.StatusCode))
		} else {
			boomer.Events.Publish("request_success", "get", url, elapsed, resp.ContentLength)
		}
	}
}

func httpReq(method string, url string, bodysize int64, headers []header) func() {
	file := postData[:bodysize]
	return func() {
		var req *http.Request

		start := boomer.Now()

		// if filename != "" {
		// 	file, _ = os.Open(filename)
		// } else {
		// 	file = nil
		// }

		req, _ = http.NewRequest(method, url, bytes.NewReader(file))

		if headers != nil {
			for _, header := range headers {
				req.Header.Set(header.name, header.value)
			}
			log.Println("in headers")
		}

		resp, err := http.DefaultClient.Do(req)
		elapsed := boomer.Now() - start
		if elapsed < 0 {
			elapsed = 0
		}
		if err != nil {
			log.Println(err)
			boomer.Events.Publish("request_failure", method, url, elapsed, err.Error())
		} else {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				boomer.Events.Publish("request_failure", method, url, elapsed, strconv.Itoa(resp.StatusCode))
			} else {
				boomer.Events.Publish("request_success", method, url, elapsed, resp.ContentLength)
				log.Println(string(body))
			}
		}

	}
}

func WeightFn(params weightParams) func() int {
	return func() (weight int) {
		base := 0.0
		if params.frequency != 0 {
			base = math.Cos(float64(time.Now().Unix())*(2*math.Pi/float64(params.frequency)) + float64(params.phase))
		}
		weight = int(base*float64(params.magnitude)) + params.constant
		if weight < 0 {
			weight = 0
		}
		return
	}
}

func getTaskParams(testDefinition Test) *boomer.Task {
	fn := httpReq(testDefinition.Method, testDefinition.Url, testDefinition.Body, testDefinition.Headers)
	weightFn := WeightFn(testDefinition.Weight)
	task := &boomer.Task{
		Name:     testDefinition.Url,
		WeightFn: weightFn,
		Fn:       fn,
	}
	//taskJson, _ := json.Marshal(task)
	log.Println(testDefinition.Method, testDefinition.Url, testDefinition.Body)
	return task
}

func main() {
	log.Println("Executing main function")
	rawTestDefinitions, _ := ioutil.ReadFile(testDefinitionsFile)
	log.Println("FileContent", string(rawTestDefinitions))
	var testDefinitions []Test
	err := json.Unmarshal(rawTestDefinitions, &testDefinitions)
	if err != nil {
		log.Println(err.Error())
	}
	var taskList []*boomer.Task

	for i, testDefinition := range testDefinitions {
		log.Println(i)
		log.Println(testDefinition.Method, testDefinition.Url, testDefinition.Body)
		taskList = append(taskList, getTaskParams(testDefinition))
	}

	boomer.Run(taskList...)
}

func init() {
	maxIdleConnections, _ = strconv.Atoi(os.Getenv("MAX_IDLE_CONNECTIONS"))
	log.Println("MaxIdleConnections", maxIdleConnections)
	testDefinitionsFile = os.Getenv("TEST_DEFINITIONS")
	log.Println("TestDefinition File", testDefinitionsFile)
	suiteName = strings.Split(os.Getenv("SUITE_NAME"), ",")
	log.Println("SuiteName", suiteName)
	maxBodySize, _ := strconv.Atoi(os.Getenv("MAX_BODY_SIZE"))
	postData = make([]byte, maxBodySize)
	rand.Read(postData)
	log.Println("maxBodySize", maxBodySize)
	log.Println("Data", string(postData[:100]))
}
