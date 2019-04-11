// Copyright (c) 2017-2019 The massnet developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"massnet.org/mass-wallet/version"

	"massnet.org/mass-wallet/config"

	"github.com/btcsuite/winsvc/eventlog"
	"github.com/btcsuite/winsvc/mgr"
	"github.com/btcsuite/winsvc/svc"
)

const (
	svcName = "masssvc"

	svcDisplayName = "Mass Service"

	svcDesc = "Downloads and stays synchronized with the mass " +
		"chain and provides chain services to applications."
)

var elog *eventlog.Log

func logServiceStartOfDay(srvr *server) {
	var message string
	message += fmt.Sprintf("Version %s\n", version.GetVersion())
	message += fmt.Sprintf("Configuration directory: %s\n", config.MassHomeDir)
	message += fmt.Sprintf("Configuration file: %s\n", cfg.ConfigFile)
	message += fmt.Sprintf("Data directory: %s\n", cfg.Db.DataDir)

	elog.Info(1, message)
}

type massService struct{}

func (s *massService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	// Service start is pending.
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	doneChan := make(chan error)
	serverChan := make(chan *server)
	go func() {
		err := massMain(serverChan)
		doneChan <- err
	}()

	// Service is now started.
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	var mainServer *server
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}

				if mainServer != nil {
					mainServer.Stop()
				} else {
					break loop
				}

			default:
				elog.Error(1, fmt.Sprintf("Unexpected control "+
					"request #%d.", c))
			}

		case srvr := <-serverChan:
			mainServer = srvr
			logServiceStartOfDay(mainServer)

		case err := <-doneChan:
			if err != nil {
				elog.Error(1, err.Error())
			}
			break loop
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}

func installService() error {
	exePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}
	if filepath.Ext(exePath) == "" {
		exePath += ".exe"
	}

	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err == nil {
		service.Close()
		return fmt.Errorf("service %s already exists", svcName)
	}

	service, err = serviceManager.CreateService(svcName, exePath, mgr.Config{
		DisplayName: svcDisplayName,
		Description: svcDesc,
	})
	if err != nil {
		return err
	}
	defer service.Close()

	eventlog.Remove(svcName)
	eventsSupported := uint32(eventlog.Error | eventlog.Warning | eventlog.Info)
	err = eventlog.InstallAsEventCreate(svcName, eventsSupported)
	if err != nil {
		return err
	}

	return nil
}

func removeService() error {
	// Connect to the windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	// Ensure the service exists.
	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", svcName)
	}
	defer service.Close()

	// Remove the service.
	err = service.Delete()
	if err != nil {
		return err
	}

	return nil
}

// startService attempts to start the mass service.
func startService() error {
	// Connect to the windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	err = service.Start(os.Args)
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

func controlService(c svc.Cmd, to svc.State) error {
	// Connect to the windows service manager.
	serviceManager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer serviceManager.Disconnect()

	service, err := serviceManager.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer service.Close()

	status, err := service.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}

	// Send the control message.
	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go "+
				"to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service "+
				"status: %v", err)
		}
	}

	return nil
}

func performServiceCommand(command string) error {
	var err error
	switch command {
	case "install":
		err = installService()

	case "remove":
		err = removeService()

	case "start":
		err = startService()

	case "stop":
		err = controlService(svc.Stop, svc.Stopped)

	default:
		err = fmt.Errorf("invalid service command [%s]", command)
	}

	return err
}

func serviceMain() (bool, error) {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return false, err
	}
	if isInteractive {
		return false, nil
	}

	elog, err = eventlog.Open(svcName)
	if err != nil {
		return false, err
	}
	defer elog.Close()

	err = svc.Run(svcName, &massService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Service start failed: %v", err))
		return true, err
	}

	return true, nil
}

// Set windows specific functions to real functions.
func init() {
	config.RunServiceCommand = performServiceCommand
	winServiceMain = serviceMain
}
