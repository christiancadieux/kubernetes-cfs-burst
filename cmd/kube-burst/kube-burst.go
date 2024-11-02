package main

import (
	"context"
	"fmt"
	"github.com/christiancadieux/kubernetes-cfs-burst/pkg/burst"
	"github.com/christiancadieux/kubernetes-cfs-burst/pkg/client"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

const (
	MAX_BURST_PERCENT = "MAX_BURST_PERCENT"
	CGROUP            = "/sys/fs/cgroup/cpu,cpuacct/kubepods"
	MY_NODE_NAME      = "MY_NODE_NAME"
)

func main() {
	ctx, cxl := context.WithCancel(context.Background())
	defer cxl()
	logger := logrus.New()

	cgroupPath := CGROUP
	cgroupEnv := os.Getenv("CGROUP_PATH")
	if cgroupEnv != "" {
		cgroupPath = cgroupEnv
	}
	nodeName := os.Getenv(MY_NODE_NAME)
	if nodeName == "" {
		logger.Errorf("env-var %s is not defined", MY_NODE_NAME)
		os.Exit(1)
	}
	dryRun := true
	if os.Getenv("DRY_RUN") == "N" {
		dryRun = false
	}

	maxBurstPercent := 200
	max := os.Getenv(MAX_BURST_PERCENT)
	if max != "" {
		maxI, err := strconv.Atoi(max)
		if err != nil {
			logger.Errorf("Invalid %s", MAX_BURST_PERCENT)
			os.Exit(1)
		}
		maxBurstPercent = maxI
	}

	var err error
	kubeClient, err := client.LoadInClusterClient()
	if err != nil {
		fmt.Println("error", err)
		logger.Error(err)
		os.Exit(1)
	}

	namespaceMgr := burst.NewNamespaceManager(logger, ctx, kubeClient, nodeName, maxBurstPercent)
	burstMgr := burst.NewBurstManager(logger, ctx, dryRun, kubeClient, nodeName, namespaceMgr, cgroupPath)
	go func() {
		err := namespaceMgr.Watch()
		if err != nil {
			logger.Errorf("namespaceMgr.Watch - %v", err)
		}
	}()

	err = burstMgr.Run()
	if err != nil {
		logger.Errorf("burstMgr.Watch - %v", err)
	}
}
