package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/anuragpaliwal80/boomer/boomer"
)

var (
	httpClient          *http.Client
	maxIdleConnections  int
	testDefinitionsFile string
	postData            []byte
	keepAlive           bool
	dataArraySize       int64
	throughPutWait      int
)

const (
	RequestTimeout int = 0
)

type weightParams struct {
	magnitude int
	frequency int
	constant  int
	phase     int
}

type header struct {
	name  string
	value int64
}

type Test struct {
	Url     string       `json:"url,omitempty"`
	Headers []header     `json:"headers,omitempty"`
	Body    int64        `json:"body,omitempty"`
	Weight  weightParams `json:"weight,omitempty"`
	Method  string       `json:"method,omitempty"`
	Wait    int16        `json:"wait,omitempty"`
}

type suite struct {
	suite []Test
}

func createHTTPClient() *http.Client {
	log.Println("KEEP_ALIVE", keepAlive)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: maxIdleConnections,
			MaxIdleConns:        maxIdleConnections,
			DisableKeepAlives:   keepAlive,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}

func httpReq(method string, url string, bodysize int64, headers []header, wait int16) func() {
	//file := postData[:bodysize]
	return func() {
		var req *http.Request
		pr, pw := io.Pipe()
		go func() {
			for i := int64(0); i < bodysize/dataArraySize; i++ {
				pw.Write(postData)
				time.Sleep(time.Duration(throughPutWait) * time.Millisecond)
			}
			if bodysize%dataArraySize != 0 {
				pw.Write(postData[:(bodysize % dataArraySize)])
			}
			pw.Close()
		}()
		start := boomer.Now()
		req, _ = http.NewRequest(method, url, pr)
		if method != "GET" {
			req.Close = true
		}
		if headers != nil {
			for _, header := range headers {
				req.Header.Set(header.name, string(postData[:header.value]))
			}
			log.Println("in headers")
		}

		resp, err := httpClient.Do(req)
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
		time.Sleep(time.Duration(wait) * time.Second)
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
	fn := httpReq(testDefinition.Method, testDefinition.Url, testDefinition.Body, testDefinition.Headers, testDefinition.Wait)
	weightFn := WeightFn(testDefinition.Weight)
	task := &boomer.Task{
		Name:     testDefinition.Url,
		WeightFn: weightFn,
		Fn:       fn,
	}
	//taskJson, _ := json.Marshal(task)
	log.Println(testDefinition.Method, testDefinition.Url, testDefinition.Body, testDefinition.Wait)
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

	// Shuffle taskList
	for i := len(taskList) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		taskList[i], taskList[j] = taskList[j], taskList[i]
	}

	boomer.Run(taskList...)
}

func init() {
	maxIdleConnections, _ = strconv.Atoi(os.Getenv("MAX_IDLE_CONNECTIONS"))
	log.Println("MaxIdleConnections", maxIdleConnections)
	testDefinitionsFile = os.Getenv("TEST_DEFINITIONS")
	dataArraySize, err := strconv.ParseInt(os.Getenv("DATA_ARRAY_SIZE"), 10, 0)
	if err != nil {
		dataArraySize = 10000000
	}
	throughPutWait, err = strconv.Atoi(os.Getenv("THROUGH_PUT_WAIT"))
	if err != nil {
		throughPutWait = 0
	}
	log.Println("TestDefinition File", testDefinitionsFile)
	if ka, err := strconv.ParseBool(os.Getenv("KEEP_ALIVE")); err == nil {
		keepAlive = !ka
	} else {
		keepAlive = true
	}
	log.Println("KEEP_ALIVE", keepAlive)
	postData = make([]byte, dataArraySize)
	httpClient = createHTTPClient()
}
