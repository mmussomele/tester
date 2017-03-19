package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/quilt/quilt/api"
	"github.com/quilt/quilt/api/client/getter"
	"github.com/quilt/quilt/db"

	log "github.com/Sirupsen/logrus"
)

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

	if test(containers) {
		log.Info("PASSED")
	} else {
		log.Info("FAILED")
	}
}

func test(containers []db.Container) bool {
	passed := true
	for _, c := range containers {
		if !strings.Contains(c.Image, "etcd") {
			continue
		}

		fmt.Printf("Checking etcd health from %s\n", c.StitchID)
		out, err := exec.Command("quilt", "ssh", c.StitchID,
			"etcdctl", "cluster-health").CombinedOutput()
		fmt.Println(string(out))
		if err != nil || !strings.Contains(string(out), "cluster is healthy") {
			log.WithError(err).Error("FAILED, cluster is unhealthy")
			passed = false
		}
	}
	return passed
}
