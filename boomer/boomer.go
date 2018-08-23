package boomer

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// Run accepts a slice of Task and connects
// to a locust master.
func Run(tasks ...*Task) {

	// support go version below 1.5
	runtime.GOMAXPROCS(runtime.NumCPU())

	if !flag.Parsed() {
		flag.Parse()
	}

	if *runTasks != "" {
		// Run tasks without connecting to the master.
		taskNames := strings.Split(*runTasks, ",")
		for _, task := range tasks {
			if task.Name == "" {
				continue
			} else {
				for _, name := range taskNames {
					if name == task.Name {
						log.Println("Running " + task.Name)
						task.Fn()
					}
				}
			}
		}
		return
	}

	if maxRPS > 0 {
		log.Println("Max RPS that boomer may generate is limited to", maxRPS)
		maxRPSEnabled = true
	}

	var r *runner
	client := newClient()
	r = &runner{
		tasks:  tasks,
		client: client,
		nodeID: getNodeID(),
	}

	Events.Subscribe("boomer:quit", r.onQuiting)

	r.getReady()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT)

	<-c
	Events.Publish("boomer:quit")

	// wait for quit message is sent to master
	<-disconnectedFromMaster
	log.Println("shut down")

}

var runTasks *string
var maxRPS int64
var maxRPSThreshold int64
var maxRPSEnabled = false
var maxRPSControlChannel = make(chan bool)
var hatchTime = 1
var weightSyncSeconds int64

func init() {
	runTasks = flag.String("run-tasks", "", "Run tasks without connecting to the master, multiply tasks is separated by comma. Usually, it's for debug purpose.")
	flag.Int64Var(&maxRPS, "max-rps", 0, "Max RPS that boomer can generate.")
	flag.Int64Var(&weightSyncSeconds, "weight-sync-seconds", 120, "Time in seconds after weights for tasks are evaluated again")
	hatchTime, _ = strconv.Atoi(os.Getenv("HATCH_TIME_SEC"))
}
