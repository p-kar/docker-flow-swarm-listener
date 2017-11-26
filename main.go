package main

import (
	"./metrics"
	"./service"
	"golang.org/x/net/context"
)

func main() {
	logPrintf("Starting Docker Flow Swarm Listener (now with Docker events)")

	ctx := context.Background()
	s := service.NewServiceFromEnv()
	n := service.NewNotificationFromEnv()
	serve := NewServe(s, n)

	go serve.Run()

	eventStream, err := s.GetEventStream()

	if err != nil {
		metrics.RecordError("GetEventStream")
	}

	if len(n.CreateServiceAddr) > 0 {
		for {
			select {
			case msg := <-eventStream:
				err := EventHandler(msg.Actor.ID, msg.Action, s, n)
				if err != nil {
					metrics.RecordError("eventHandler")
				}
			case <-ctx.Done():
				logPrintf("Context finished")
				break
			}
		}
	}
	logPrintf("service terminating")
}
