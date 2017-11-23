package main

import (
	"./metrics"
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


func eventHandler(event_id string, action_type string, s *(service.Service), n *(service.Notification), m map[string](*[]service.SwarmService)) error{
	args := getArgs()
	swarm_service, err := s.GetServiceForEventID(event_id)
	//fmt.Printf("swarm_service_id = %s",swarm_service[0].ID)
	//log.Println("Came here successfully")
	/*for s := range swarm_service {
		//swarmServices = append(swarmServices, SwarmService{s})
		log.Println(s.ID)
	}*/
	if action_type == "create" {
		m[event_id] = swarm_service
		log.Println("Service created notif should've been sent")
		err = n.ServicesCreate(
				swarm_service,
				args.Retry,
				args.RetryInterval,
				)	
		if err != nil { 
			metrics.RecordError("ServicesCreate")
		}
	}
	if action_type == "update" {
		log.Println("Service update notif should've been sent")
	}
	if action_type == "remove" {
		log.Println("Service removed notif should've been sent")
		swarm_service = m[event_id]
		delete(m, event_id)
		err = n.ServicesRemove(&[]string{event_id}, args.Retry, args.RetryInterval)
		metrics.RecordService(len(service.CachedServices))
		if err != nil {
			metrics.RecordError("ServicesRemove")
		}
	}
	return err
}

func main() {
	log.Println("Starting Docker Flow Swarm Listener (now with Docker events)")

	ctx := context.Background()
	s := service.NewServiceFromEnv()
	n := service.NewNotificationFromEnv()
	serve := NewServe(s, n)
	go serve.Run()
	m := make(map[string](*[]service.SwarmService))
	cli := s.DockerClient
	filters := filters.NewArgs()
	filters.Add("type", "service")
	eventStream, _ := cli.Events(ctx, types.EventsOptions{Filters: filters,})

	if len(n.CreateServiceAddr) > 0 {
		for {
			select {
			case msg := <-eventStream:
				fmt.Printf("Time: %s\tType: %s\tAction: %s\tActor ID: %s\n", time.Unix(msg.Time, 0), msg.Type, msg.Action, msg.Actor.ID)
				err := eventHandler(msg.Actor.ID, msg.Action, s, n, m)
				if err != nil {
					log.Println("Awesome!")
				}
			case <-ctx.Done():
				log.Println("context done in main loop, exiting")
				break
			}
		}
	}
	log.Println("service terminating")
}
