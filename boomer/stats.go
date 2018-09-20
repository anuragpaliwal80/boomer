package boomer

import (
	"time"
)

type requestStats struct {
	entries   map[string]*statsEntry
	errors    map[string]*statsError
	total     *statsEntry
	startTime int64
}

type RequestInsights struct{
	DnsLookup int64
	TcpConnection int64
	TlsHandshake int64
	ServerProcessing int64
	NameLookup int64
	Connect int64
	PreTransfer int64
	StartTransfer int64
	ElapsedTime int64
}


func newRequestStats() *requestStats {
	entries := make(map[string]*statsEntry)
	errors := make(map[string]*statsError)

	requestStats := &requestStats{
		entries: entries,
		errors:  errors,
	}

	requestStats.total = &statsEntry{
		name:   "Total",
		method: "",
	}
	requestStats.total.reset()

	return requestStats
}

func (s *requestStats) logRequest(method, name string, responseTime RequestInsights, contentLength int64) {
	s.total.log(responseTime, contentLength)
	s.get(name, method).log(responseTime, contentLength)
}

func (s *requestStats) logError(method, name, err string) {
	s.total.logError(err)
	s.get(name, method).logError(err)

	// store error in errors map
	key := MD5(method, name, err)
	entry, ok := s.errors[key]
	if !ok {
		entry = &statsError{
			name:   name,
			method: method,
			error:  err,
		}
		s.errors[key] = entry
	}
	entry.occured()
}

func (s *requestStats) get(name string, method string) (entry *statsEntry) {
	entry, ok := s.entries[name+method]
	if !ok {
		newEntry := &statsEntry{
			name:          name,
			method:        method,
			numReqsPerSec: make(map[int64]int64),
		}
		newEntry.reset()
		s.entries[name+method] = newEntry
		return newEntry
	}
	return entry
}

func (s *requestStats) clearAll() {
	s.total = &statsEntry{
		name:   "Total",
		method: "",
	}
	s.total.reset()

	s.entries = make(map[string]*statsEntry)
	s.errors = make(map[string]*statsError)
	s.startTime = time.Now().Unix()
}

func (s *requestStats) serializeStats() []interface{} {
	entries := make([]interface{}, 0, len(s.entries))
	for _, v := range s.entries {
		if !(v.numRequests == 0 && v.numFailures == 0) {
			entries = append(entries, v.getStrippedReport())
		}
	}
	return entries
}

func (s *requestStats) serializeErrors() map[string]map[string]interface{} {
	errors := make(map[string]map[string]interface{})
	for k, v := range s.errors {
		errors[k] = v.toMap()
	}
	return errors
}

type requestInsightEntry struct {
	totalTime    int64
	minTime      int64
	maxTime      int64
	times        map[int64]int64
}

type statsEntry struct {
	name                 string
	method               string
	numRequests          int64
	numFailures          int64
	numReqsPerSec        map[int64]int64
	responseTime requestInsightEntry
	dnsLookup requestInsightEntry
	tcpConnection requestInsightEntry
	tlsHandshake requestInsightEntry
	serverProcessing requestInsightEntry
	nameLookup requestInsightEntry
	connect requestInsightEntry
	preTransfer requestInsightEntry
	startTransfer requestInsightEntry
	totalContentLength   int64
	startTime            int64
	lastRequestTimestamp int64
}

func (s *requestInsightEntry) reset() {
	s.totalTime = 0
	s.times = make(map[int64]int64)
	s.minTime = 0
	s.maxTime = 0
}

func (s *statsEntry) reset() {
	s.startTime = time.Now().Unix()
	s.numRequests = 0
	s.numFailures = 0
	s.responseTime.reset()
	s.dnsLookup.reset()
	s.tcpConnection.reset()
	s.tlsHandshake.reset()
	s.serverProcessing.reset()
	s.nameLookup.reset()
	s.connect.reset()
	s.preTransfer.reset()
	s.startTransfer.reset()
	s.lastRequestTimestamp = time.Now().Unix()
	s.numReqsPerSec = make(map[int64]int64)
	s.totalContentLength = 0
}

func (s *statsEntry) log(responseTime RequestInsights, contentLength int64) {
	s.numRequests++

	s.logTimeOfRequest()
	s.logResponseTime(responseTime)

	s.totalContentLength += contentLength
}

func (s *statsEntry) logTimeOfRequest() {
	now := time.Now().Unix()

	_, ok := s.numReqsPerSec[now]
	if !ok {
		s.numReqsPerSec[now] = 1
	} else {
		s.numReqsPerSec[now]++
	}

	s.lastRequestTimestamp = now
}

func (entry *requestInsightEntry) updateTime(time int64){
	entry.totalTime += time
	if entry.minTime == 0 {
		entry.minTime = time
	}

	if time < entry.minTime {
		entry.minTime = time
	}

	if time > entry.maxTime {
		entry.maxTime = time
	}
	roundedResponseTime := int64(0)

	// to avoid to much data that has to be transferred to the master node when
	// running in distributed mode, we save the response time rounded in a dict
	// so that 147 becomes 150, 3432 becomes 3400 and 58760 becomes 59000
	// see also locust's stats.py
	if time < 100 {
		roundedResponseTime = time
	} else if time < 1000 {
		roundedResponseTime = int64(round(float64(time), .5, -1))
	} else if time < 10000 {
		roundedResponseTime = int64(round(float64(time), .5, -2))
	} else {
		roundedResponseTime = int64(round(float64(time), .5, -3))
	}

	_, ok := entry.times[roundedResponseTime]
	if !ok {
		entry.times[roundedResponseTime] = 1
	} else {
		entry.times[roundedResponseTime]++
	}
}

func (s *statsEntry) logResponseTime(responseTime RequestInsights) {
		s.responseTime.updateTime(responseTime.ElapsedTime)
		s.dnsLookup.updateTime(responseTime.DnsLookup)
		s.connect.updateTime(responseTime.Connect)
		s.tcpConnection.updateTime(responseTime.TcpConnection)
		s.tlsHandshake.updateTime(responseTime.TlsHandshake)
		s.serverProcessing.updateTime(responseTime.ServerProcessing)
		s.nameLookup.updateTime(responseTime.NameLookup)
		s.preTransfer.updateTime(responseTime.PreTransfer)
		s.startTransfer.updateTime(responseTime.StartTransfer)
}

func (s *statsEntry) logError(err string) {
	s.numFailures++
}

func (s *requestInsightEntry) serialize() map[string]interface{} {
	result := make(map[string]interface{})
	result["total_response_time"] = s.totalTime
	result["max_response_time"] = s.maxTime
	result["min_response_time"] = s.minTime
	return result
}

func (s *statsEntry) serialize() map[string]interface{} {
	result := make(map[string]interface{})
	result["name"] = s.name
	result["method"] = s.method
	result["last_request_timestamp"] = s.lastRequestTimestamp
	result["start_time"] = s.startTime
	result["num_requests"] = s.numRequests
	result["num_failures"] = s.numFailures
	result["total_response_time"] = s.responseTime.totalTime
	result["max_response_time"] = s.responseTime.maxTime
	result["min_response_time"] = s.responseTime.minTime
	result["total_content_length"] = s.totalContentLength
	result["dns_lookup"] = s.dnsLookup.serialize()
	result["tcp_connection"] = s.tcpConnection.serialize()
	result["tls_handshake"] = s.tlsHandshake.serialize()
	result["server_processing"] = s.serverProcessing.serialize()
	result["name_lookup"] = s.nameLookup.serialize()
	result["connect"] = s.connect.serialize()
	result["pre_tranfer"] = s.preTransfer.serialize()
	result["start_tranfer"] = s.startTransfer.serialize()
	result["response_times"] = s.responseTime.times
	result["num_reqs_per_sec"] = s.numReqsPerSec
	return result
}

func (s *statsEntry) getStrippedReport() map[string]interface{} {
	report := s.serialize()
	s.reset()
	return report
}

type statsError struct {
	name       string
	method     string
	error      string
	occurences int64
}

func (err *statsError) occured() {
	err.occurences++
}

func (err *statsError) toMap() map[string]interface{} {
	m := make(map[string]interface{})

	m["method"] = err.method
	m["name"] = err.name
	m["error"] = err.error
	m["occurences"] = err.occurences

	return m
}

func collectReportData() map[string]interface{} {
	data := make(map[string]interface{})

	data["stats"] = stats.serializeStats()
	data["stats_total"] = stats.total.getStrippedReport()
	data["errors"] = stats.serializeErrors()

	stats.errors = make(map[string]*statsError)

	return data
}

type requestSuccess struct {
	requestType    string
	name           string
	requestIns   RequestInsights
	responseLength int64
}

type requestFailure struct {
	requestType  string
	name         string
	responseTime int64
	error        string
}

var stats = newRequestStats()
var requestSuccessChannel = make(chan *requestSuccess, 1000)
var requestFailureChannel = make(chan *requestFailure, 1000)
var clearStatsChannel = make(chan bool)
var messageToRunner = make(chan map[string]interface{}, 100)

func init() {
	stats.entries = make(map[string]*statsEntry)
	stats.errors = make(map[string]*statsError)
	go func() {
		var ticker = time.NewTicker(slaveReportInterval)
		for {
			select {
			case m := <-requestSuccessChannel:
				stats.logRequest(m.requestType, m.name, m.requestIns, m.responseLength)
			case n := <-requestFailureChannel:
				stats.logError(n.requestType, n.name, n.error)
			case <-clearStatsChannel:
				stats.clearAll()
			case <-ticker.C:
				data := collectReportData()
				// send data to channel, no network IO in this goroutine
				messageToRunner <- data
			}
		}
	}()
}
