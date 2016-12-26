package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
)

type schedstat struct {
	YldCount    float64
	SchedCount  float64
	SchedGoidle float64
	TtwuCount   float64
	TtwuLocal   float64
	CpuTime     float64
	RunDelay    float64
	Pcount      float64
}

var graphDef = map[string]mp.Graphs{
	"schedstat.cpu.#": {
		Label: "CPU Schedstat",
		Unit:  "float",
		Metrics: []mp.Metrics{
			{Name: "run_delay", Label: "run_delay", Diff: false, Stacked: false},
			{Name: "cpu_time", Label: "cpu_time", Diff: false, Stacked: false},
		},
	},
}

func collectSchedstat() (map[string]*schedstat, error) {
	bytes, err := ioutil.ReadFile("/proc/schedstat")
	if err != nil {
		return nil, err
	}
	return parseProcSchedstat(string(bytes))
}

func parseProcSchedstat(str string) (map[string]*schedstat, error) {
	var stats = make(map[string]*schedstat)
	var err error

	for _, line := range strings.Split(str, "\n") {
		if strings.HasPrefix(line, "cpu") {
			fields := strings.Fields(line)
			key := fields[0]

			values := make([]float64, 10)
			for i, strValue := range fields[1:] {
				values[i], err = strconv.ParseFloat(strValue, 64)
				if err != nil {
					return nil, err
				}
			}

			ps := &schedstat{
				// cpuN:     filledValues[0]
				YldCount: values[0],
				// 0:        filledValues[1]
				SchedCount:  values[2],
				SchedGoidle: values[3],
				TtwuCount:   values[4],
				TtwuLocal:   values[5],
				CpuTime:     values[6], /* jiffies */
				RunDelay:    values[7], /* jiffies */
				Pcount:      values[8], /* number of processes */
			}

			stats[key] = ps
		}
	}

	return stats, nil
}

func printSchedstat(stats map[string]*schedstat) {
	// todo: jiffies
	user_hz := 1000.0

	now := time.Now()
	for i, s := range stats {
		fmt.Printf("schedstat.cpu.%s.run_delay\t%f\t%d\n", i, s.RunDelay/user_hz/user_hz/1000*100, now.Unix())
		fmt.Printf("schedstat.cpu.%s.cpu_time\t%f\t%d\n", i, s.CpuTime/user_hz/user_hz/1000*100, now.Unix())
	}
}

func printDefinitions() {
	fmt.Println("# mackerel-agent-plugin")
	var graphs mp.GraphDef
	graphs.Graphs = graphDef

	b, err := json.Marshal(graphs)
	if err != nil {
		log.Fatalln("OutputDefinitions: ", err)
	}
	fmt.Println(string(b))
}

func main() {

	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") != "" {
		printDefinitions()
	} else {
		interval := 1000 * time.Millisecond
		prevStats, _ := collectSchedstat()

		time.Sleep(interval)

		currentStats, _ := collectSchedstat()
		for k, _ := range prevStats {
			currentStats[k].RunDelay -= prevStats[k].RunDelay
			currentStats[k].CpuTime -= prevStats[k].CpuTime
		}

		printSchedstat(currentStats)
	}
}
