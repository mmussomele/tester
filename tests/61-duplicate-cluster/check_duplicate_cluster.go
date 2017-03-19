package main

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/quilt/quilt/api"
	"github.com/quilt/quilt/api/client/getter"
)

func main() {
	clientGetter := getter.New()

	clnt, err := clientGetter.Client(api.DefaultSocket)
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't get quiltctl client.")
	}
	defer clnt.Close()

	leader, err := clientGetter.LeaderClient(clnt)
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't get leader client.")
	}
	defer leader.Close()

	containers, err := leader.QueryContainers()
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't query containers.")
	}

	psPretty, err := exec.Command("quilt", "ps").Output()
	if err != nil {
		log.WithError(err).Fatal("FAILED, `quilt ps` failed.")
	}
	fmt.Println("`quilt ps` output:")
	fmt.Println(string(psPretty))

	var masters []string
	totalWorkers := 0
	for _, dbc := range containers {
		if strings.Join(dbc.Command, " ") == "run master" {
			id := dbc.StitchID
			masters = append(masters, id)
		} else {
			totalWorkers++
		}
	}
	if len(masters) != 2 {
		log.WithField("masters", masters).Fatal(
			"FAILED, expected 2 Spark masters.")
	}

	failed := false
	for _, master := range masters {
		logs, err := exec.Command("quilt", "logs", master).CombinedOutput()
		if err != nil {
			log.WithError(err).Fatal(
				"FAILED, unable to get Spark master logs.")
		}

		// Each cluster's workers should connect only to its own master.
		logsStr := string(logs)
		workerCount := strings.Count(logsStr, "Registering worker")
		if workerCount != totalWorkers/2 {
			failed = true
			log.WithFields(log.Fields{
				"master":                master,
				"worker count":          workerCount,
				"expected worker count": totalWorkers / 2,
			}).Error("FAILED, wrong number of workers connected to master")
		}

		fmt.Printf("`quilt logs %s` output:\n", master)
		fmt.Println(logsStr)
	}

	if !failed {
		fmt.Println("PASSED")
	}
}
