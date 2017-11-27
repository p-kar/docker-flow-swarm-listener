package service

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"golang.org/x/net/context"
	"../metrics"
)

func ProcessEventStream(s *Service, n *Notification, Interval int, Retry int, RetryInterval int) error{
	ctx := context.Background()
	filters := filters.NewArgs()
	filters.Add("type", "service")
	eventStream, err := s.DockerClient.Events(
		ctx, 
		types.EventsOptions{Filters: filters,},
	)

	if err != nil {
		metrics.RecordError("GetEventStream")
	}
	var err_event error
	if len(n.CreateServiceAddr) > 0 {
		for {
			select {
			case msg := <-eventStream:
				err_event := EventHandler(msg.Actor.ID, msg.Action, s, n, Interval, Retry, RetryInterval)
				if err_event != nil {
					metrics.RecordError("eventHandler")
				}
			case <-ctx.Done():
				logPrintf("Context finished")
				break
			}
		}
	}
	return err_event
}