package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/quilt/quilt/api"
	"github.com/quilt/quilt/api/client/getter"
	"github.com/quilt/quilt/db"

	log "github.com/Sirupsen/logrus"
)

// Response represents the JSON response from the quilt/mean-service
type Response struct {
	Text string `json:"text"`
	ID   string `json:"_id"`
	V    int    `json:"__v"`
}

func main() {
	clientGetter := getter.New()

	clnt, err := clientGetter.Client(api.DefaultSocket)
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't get quiltctl client")
	}
	defer clnt.Close()

	leader, err := clientGetter.LeaderClient(clnt)
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't get leader client")
	}

	containers, err := leader.QueryContainers()
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't query containers")
	}

	machines, err := clnt.QueryMachines()
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't query machines")
	}

	publicIPs := getPublicIPs(machines, containers)
	fmt.Printf("Public IPs: %s\n", publicIPs)
	if len(publicIPs) == 0 {
		log.Fatal("FAILED, Found no public IPs")
	}

	logTest := logContainers(containers)
	getTest := httpGetTest(publicIPs)
	postTest := httpPostTest(publicIPs)

	if logTest && getTest && postTest {
		fmt.Println("PASSED")
	} else {
		fmt.Println("FAILED")
	}
}

func logContainers(containers []db.Container) bool {
	var failed bool
	for _, c := range containers {
		out, err := exec.Command("quilt", "logs", c.StitchID).CombinedOutput()
		if err != nil {
			log.WithError(err).Errorf("Failed to log: %s", c)
			failed = true
		}
		fmt.Printf("Container: %s\n%s\n\n", c, string(out))
	}

	return !failed
}

func getPublicIPs(machines []db.Machine, containers []db.Container) []string {

	minionIPMap := map[string]string{}
	for _, m := range machines {
		minionIPMap[m.PrivateIP] = m.PublicIP
	}
	var publicIPs []string
	for _, c := range containers {
		if strings.Contains(c.Image, "haproxy") {
			ip, ok := minionIPMap[c.Minion]
			if !ok {
				log.WithField("container", c).Fatal(
					"FAILED, HAProxy with no public IP")
			}
			publicIPs = append(publicIPs, ip)
		}
	}

	return publicIPs
}

func httpGetTest(publicIPs []string) bool {

	fmt.Println("HTTP Get Test")

	var failed bool
	for i := 0; i < 25; i++ {
		for _, ip := range publicIPs {
			resp, err := http.Get("http://" + ip)
			if err != nil {
				log.WithError(err).Error("HTTP Error")
				failed = true
				continue
			}

			if resp.StatusCode != 200 {
				failed = true
				fmt.Println(resp)
			}
		}
	}

	return !failed
}

// checkInstances queries the todos for each instance and makes sure that all
// data is available from each instance.
func checkInstances(publicIPs []string, expectedTodos int) bool {
	var todos []Response
	failed := false
	for _, ip := range publicIPs {
		endpoint := fmt.Sprintf("http://%s/api/todos", ip)
		resp, err := http.Get(endpoint)
		if err != nil {
			log.WithError(err).Error("HTTP Error")
			failed = true
			continue
		}

		if resp.StatusCode != 200 {
			failed = true
			fmt.Println(resp)
		}

		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&todos)
		if err != nil {
			log.WithField("ip", ip).WithError(err).Error(
				"JSON decoding error")
			failed = true
			continue
		}

		defer resp.Body.Close()
		if len(todos) != expectedTodos {
			log.WithFields(log.Fields{
				"ip":       ip,
				"expected": expectedTodos,
				"actual":   len(todos),
			}).Error(
				"POSTed data not consistent for endpoint")
			failed = true
			continue
		}
	}

	return !failed
}

// httpPostTest tests that data persists across the quilt/mean-service.
// Data is POSTed to each instance, and then we check from all instances that
// all of the data can be recovered.
func httpPostTest(publicIPs []string) bool {

	fmt.Println("HTTP Post Test")

	var failed bool
	for i := 0; i < 10; i++ {
		for _, ip := range publicIPs {
			endpoint := fmt.Sprintf("http://%s/api/todos", ip)

			jsonStr := fmt.Sprintf("{\"text\": \"%s-%d\"}", ip, i)
			jsonBytes := bytes.NewBufferString(jsonStr)

			resp, err := http.Post(endpoint, "application/json", jsonBytes)
			if err != nil {
				log.WithError(err).Error("HTTP Error")
				failed = true
				continue
			}

			if resp.StatusCode != 200 {
				failed = true
				fmt.Println(resp)
			}
		}
	}

	return checkInstances(publicIPs, 10*len(publicIPs)) && !failed
}
