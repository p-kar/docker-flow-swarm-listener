package main 

import (
	"./metrics"
	"./service"
)

func EventHandler(event_id string, action_type string, s *(service.Service), n *(service.Notification)) error{
	args := getArgs()
	swarm_service, err := s.GetServiceForEventID(event_id)
	
	if action_type == "create" {
		
		newServices, err := s.GetNewServices(swarm_service)
		if err != nil {
				metrics.RecordError("GetNewServices")
		}
		err = n.ServicesCreate(
				newServices,
				args.Retry,
				args.RetryInterval,
				)	
		if err != nil { 
			metrics.RecordError("ServicesCreate")
		}
	}
	/*
	if action_type == "update" {
		logPrintf("Service update to be sent")
	}
	*/
	if action_type == "remove" {

		removedServices := s.GetRemovedServices(swarm_service)
		err = n.ServicesRemove(removedServices, args.Retry, args.RetryInterval)
		metrics.RecordService(len(service.CachedServices))
		if err != nil {
			metrics.RecordError("ServicesRemove")
		}
	}
	return err
}