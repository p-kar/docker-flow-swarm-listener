package main

import (
	// "./metrics"
	"./service"
	"time"
	// for log formatting
	// "encoding/json"
	"fmt"
	// "io"
	"log"
	//"os"
	// Docker events API
	//"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	// "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"golang.org/x/net/context"
)

/*
func decodeEvents(body io.Reader) {
	dec := json.NewDecoder(body)
	for {
		var event events.Message
		err := dec.Decode(&event)
		if err != nil && err == io.EOF {
			break
		}
		log.Println(event)
	}
}
*/

func main() {
	log.Println("Starting Docker Flow Swarm Listener (now with Docker events)")

	filters := filters.NewArgs()
	filters.Add("type", "service")
	/*
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	var dockerApiVersion string = "v1.22"
	host := "unix:///var/run/docker.sock"
	if len(os.Getenv("DF_DOCKER_HOST")) > 0 {
		host = os.Getenv("DF_DOCKER_HOST")
	}
	cli, err := client.NewClient(host, dockerApiVersion, nil, defaultHeaders)
	if err != nil {
		panic(err)
	}
	*/
	s := service.NewServiceFromEnv()
	cli := s.DockerClient

	ctx := context.Background()
	// eventStream, errs := cli.Events(ctx, types.EventsOptions{Filters: filters,})
	eventStream, _ := cli.Events(ctx, types.EventsOptions{Filters: filters,})

	for {
		select {
		case msg := <-eventStream:
			fmt.Printf("Time: %s\tType: %s\tAction: %s\tActor ID: %s\n", time.Unix(msg.Time, 0), msg.Type, msg.Action, msg.Actor.ID)
			// log.Println(event)
			// decodeEvents(event)
		case <-ctx.Done():
			log.Println("context done in main loop, exiting")
			break
		/*
		case err := <-errs:
			if err == io.EOF {
				break
			}
			log.Println("main loop error")
		*/
		}
	}
	log.Println("service terminating")
}
