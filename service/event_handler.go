package service 

import (
	"../metrics"
)

func EventHandler(event_id string, action_type string, s *(Service), n *(Notification), Interval int, Retry int, RetryInterval int) error{
	
	swarm_service, err := s.GetServiceForEventID(event_id)
	
	if action_type == "create" {
		
		newServices, err := s.GetNewServices(swarm_service)
		if err != nil {
				metrics.RecordError("GetNewServices")
		}
		err = n.ServicesCreate(
				newServices,
				Retry,
				RetryInterval,
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
		err = n.ServicesRemove(removedServices, Retry, RetryInterval)
		metrics.RecordService(len(CachedServices))
		if err != nil {
			metrics.RecordError("ServicesRemove")
		}
	}
	return err
}