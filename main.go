package main

import (
	"./service"
	"./metrics"
)

func main() {
	logPrintf("Starting Docker Flow Swarm Listener (now with Docker events)")

	s := service.NewServiceFromEnv()
	n := service.NewNotificationFromEnv()
	serve := NewServe(s, n)

	go serve.Run()

	args := getArgs()
	err := service.ProcessEventStream(s, n, args.Interval, args.Retry, args.RetryInterval)
	
	if err != nil {
		metrics.RecordError("ProcessEventStream")
	}
	logPrintf("service terminating")
}
