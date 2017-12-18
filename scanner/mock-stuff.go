package scanner

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// need: mock hub, ?mock apiserver?

// MockHub is a mock implementation of ScanClientInterface .
type MockHub struct {
	inProgressImages []string
	finishedImages   map[string]int
}

func NewMockHub() *MockHub {
	hub := new(MockHub)
	hub.inProgressImages = []string{}
	hub.finishedImages = make(map[string]int)
	return hub
}

func (hub *MockHub) startRandomScanFinishing() {
	fmt.Println("starting!")
	for {
		time.Sleep(3 * time.Second)
		// TODO should lock the hub
		length := len(hub.inProgressImages)
		fmt.Println("in progress -- [", strings.Join(hub.inProgressImages, ", "), "]")
		if length <= 0 {
			continue
		}
		index := rand.Intn(length)
		image := hub.inProgressImages[index]
		fmt.Println("something finished --", image)
		hub.inProgressImages = append(hub.inProgressImages[:index], hub.inProgressImages[index+1:]...)
		hub.finishedImages[image] = 1
	}
}

func (hub *MockHub) GetFinishedJobs() []ScanJob {
	images := []ScanJob{}
	// TODO should lock hub
	// TODO uncomment
	/*
		for image := range hub.finishedImages {
			images = append(images, image)
		}
	*/
	hub.finishedImages = make(map[string]int)
	return images
}

func (hub *MockHub) Scan(job ScanJob) {
	// TODO implement
	/*
		for _, image := range images {
			hub.inProgressImages = append(hub.inProgressImages, image)
		}
	*/
}
