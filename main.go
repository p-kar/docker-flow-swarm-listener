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

func main() {
	log.Println("Starting Docker Flow Swarm Listener (now with Docker events)")

	ctx := context.Background()
	s := service.NewServiceFromEnv()
	n:= service.NewNotificationFromEnv()
	serve := NewServe(s, n)
	go serve.Run()

	cli := s.DockerClient
	filters := filters.NewArgs()
	filters.Add("type", "service")
	eventStream, _ := cli.Events(ctx, types.EventsOptions{Filters: filters,})

	if len(n.CreateServiceAddr) > 0 {
		for {
			select {
			case msg := <-eventStream:
				fmt.Printf("Time: %s\tType: %s\tAction: %s\tActor ID: %s\n", time.Unix(msg.Time, 0), msg.Type, msg.Action, msg.Actor.ID)
			case <-ctx.Done():
				log.Println("context done in main loop, exiting")
				break
			}
		}
	}
	log.Println("service terminating")
}
