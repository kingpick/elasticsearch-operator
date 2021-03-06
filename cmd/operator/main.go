/*
Copyright (c) 2017, UPMC Enterprises
All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name UPMC Enterprises nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL UPMC ENTERPRISES BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/upmc-enterprises/elasticsearch-operator/pkg/controller"
	"github.com/upmc-enterprises/elasticsearch-operator/pkg/processor"
	"github.com/upmc-enterprises/elasticsearch-operator/util/k8sutil"
)

var (
	appVersion = "0.0.1"

	printVersion bool
	baseImage    string
	kubeCfgFile  string
	masterHost   string
	namespace    = os.Getenv("NAMESPACE")
)

func init() {
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&baseImage, "baseImage", "upmcenterprises/docker-elasticsearch-kubernetes:5.1.1", "Base image to use when spinning up the elasticsearch components.")
	flag.StringVar(&kubeCfgFile, "kubecfg-file", "", "Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens")
	flag.StringVar(&masterHost, "masterhost", "http://127.0.0.1:8001", "Full url to k8s api server")
	flag.Parse()
}

// Main entrypoint
func Main() int {
	if printVersion {
		fmt.Println("elasticsearch-operator", appVersion)
		os.Exit(0)
	}

	logrus.Info("elasticsearch operator starting up!")

	// Print params configured
	logrus.Info("Using Variables:")
	logrus.Infof("   baseImage: %s", baseImage)

	// Init
	k8sclient, err := k8sutil.New(kubeCfgFile, masterHost)
	controller, err := controller.New("elasticcluster", namespace, k8sclient)
	processor, err := processor.New(k8sclient, baseImage)

	if err != nil {
		logrus.Error("Could not init Controller! ", err)
		return 1
	}

	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	// Kick it off
	controller.Run()
	processor.Run()

	// Watch for events that add, modify, or delete ElasticSearchCluster definitions andlog
	// process them asynchronously.
	logrus.Info("Watching for elastic search events...")
	wg.Add(1)
	processor.WatchElasticSearchClusterEvents(doneChan, &wg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-signalChan:
			logrus.Error("Shutdown signal received, exiting...")
			close(doneChan)
			wg.Wait()
			os.Exit(0)
		}
	}
}

func main() {
	os.Exit(Main())
}
