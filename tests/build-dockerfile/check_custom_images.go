package main

import (
	"fmt"
	"os/exec"
	"strconv"
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
	defer leader.Close()

	containers, err := leader.QueryContainers()
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't query containers")
	}

	machines, err := clnt.QueryMachines()
	if err != nil {
		log.WithError(err).Fatal("FAILED, couldn't query machines")
	}

	// The images deployed for the given Dockerfile.
	dockerfileToImages := make(map[string][]string)

	// The number of containers deployed for each Dockerfile.
	dockerfileCount := make(map[string]int)

	for _, c := range containers {
		if !strings.Contains(c.Image, "test-custom-image") {
			continue
		}

		dockerfileID, imageID, err := getContainerInfo(c.StitchID)
		if err != nil {
			log.WithError(err).WithField("container", c).
				Fatal("FAILED, couldn't get container info")
		}

		dockerfileToImages[dockerfileID] = append(
			dockerfileToImages[dockerfileID], imageID)
		dockerfileCount[dockerfileID]++
	}

	fmt.Println("Dockerfile to image mappings:", dockerfileToImages)
	fmt.Println("Dockerfile counts:", dockerfileCount)

	reuseErr := checkReuseImage(dockerfileToImages)
	if reuseErr != nil {
		log.WithError(reuseErr).Error("FAILED")
	}

	countErr := checkImageCounts(machines, dockerfileCount)
	if reuseErr != nil {
		log.WithError(countErr).Error("FAILED")
	}

	if reuseErr == nil && countErr == nil {
		fmt.Println("PASSED")
	}
}

func checkReuseImage(dockerfileToImages map[string][]string) error {
	for dk, images := range dockerfileToImages {
		for _, otherImg := range images {
			if otherImg != images[0] {
				return fmt.Errorf("images for DockerfileID %s not "+
					"reused: %v", dk, images)
			}
		}
	}
	return nil
}

func checkImageCounts(machines []db.Machine, dockerfileCounts map[string]int) error {
	nWorker := 0
	for _, m := range machines {
		if m.Role == db.Worker {
			nWorker++
		}
	}

	for i := 0; i < nWorker; i++ {
		if actual := dockerfileCounts[strconv.Itoa(i)]; actual != 2 {
			return fmt.Errorf("DockerfileID %d had %d containers, "+
				"expected %d", i, actual, 2)
		}
	}

	return nil
}

func getContainerInfo(stitchID string) (string, string, error) {
	dockerfileID, err := exec.Command(
		"quilt", "ssh", stitchID, "cat /dockerfile-id").CombinedOutput()
	if err != nil {
		return "", "", err
	}
	imageID, err := exec.Command(
		"quilt", "ssh", stitchID, "cat /image-id").CombinedOutput()
	return strings.TrimSpace(string(dockerfileID)),
		strings.TrimSpace(string(imageID)),
		err
}
