/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package server

import (
	"context"
	"fmt"
	"net"
	"time"

	pb_vlalpha "github.ibm.com/PuDong/ibm-storage-odf-block-driver/api/pb/v1alpha"
	restclient "github.ibm.com/PuDong/ibm-storage-odf-block-driver/pkg/rest"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

//DriverConfig DriverConfig
type DriverConfig struct {
	Net  string
	Addr string
}

//Server Server
type Server struct {
	name         string
	config       *DriverConfig
	grpcserver   *grpc.Server
	listener     net.Listener
	fsrestclient *restclient.FSRestClient
}

// Create a server
func New(config *DriverConfig, restConfig *restclient.Config) (*Server, error) {
	var server *Server

	if config == nil {
		return nil, fmt.Errorf("Error, no server config specified")
	}

	if len(config.Net) == 0 {
		return nil, fmt.Errorf("Need to specify server net")
	}
	if len(config.Addr) == 0 {
		return nil, fmt.Errorf("Need to specify server address")
	}

	klog.Info("====================Starting Driver====================")

	lis, err := net.Listen(config.Net, config.Addr)
	if err != nil {
		if lis != nil {
			lis.Close()
		}
		klog.Errorf("Server listen failed: %s", err)
		return nil, err
	}

	fsclient, err := restclient.NewFSRestClient(restConfig)
	if err != nil {
		klog.Errorf("fail to init rest client, error:%s", err)
		return nil, err
	}

	gserver := grpc.NewServer()

	server = &Server{
		name:         "flash_storage_driver",
		config:       config,
		grpcserver:   gserver,
		listener:     lis,
		fsrestclient: fsclient,
	}

	return server, nil
}

//Start start
func (s *Server) Start() {
	pb_vlalpha.RegisterStorageDriverServiceServer(s.grpcserver, s)

	go func() {
		if err := s.grpcserver.Serve(s.listener); err != nil {
			klog.Errorf("failed to start server, error: %v", err)
		}
	}()
}

//Stop stop
func (s *Server) Stop(force bool) {
	klog.Info("stopping server...")

	if s == nil {
		return
	}

	if s.grpcserver != nil {
		if force {
			s.grpcserver.Stop()
		} else {
			s.grpcserver.GracefulStop()
		}
	}

	if s.listener != nil {
		s.listener.Close()
	}
}

//Cleanup Cleanup
func (s *Server) Cleanup() {
	s.Stop(false)
}

//GetStorage GetStorage
func (s *Server) GetStorage(cxt context.Context, r *pb_vlalpha.GetStorageRequest) (*pb_vlalpha.GetStorageResponse, error) {
	klog.Infof("Try to get storage info")
	bT := time.Now()
	result := &pb_vlalpha.GetStorageResponse{
		Storage: &pb_vlalpha.Storage{
			Id:           "Id",
			Name:         "name",
			Version:      "version",
			MaxCapacity:  "0",
			UsedCapacity: "0",
		},
	}
	result.Storage.State = pb_vlalpha.StorageState_StorageAvailable

	if s.fsrestclient != nil {
		var storagesystem restclient.StorageSystem
		storagesystem, err := s.fsrestclient.Lssystem()
		if err != nil {
			err = s.fsrestclient.Reconnect()
			if err != nil {
				klog.Errorf("fails to authenticate rest server, err:%s", err)
				return nil, err
			}
		}
		totalfree, ok := storagesystem["total_free_space"]

		if ok {
			result.Storage.MaxCapacity = fmt.Sprintf("%v", totalfree)
		}

		totalused, ok := storagesystem["total_used_capacity"]
		if ok {
			result.Storage.UsedCapacity = fmt.Sprintf("%v", totalused)
		}

		name, ok := storagesystem["name"]
		if ok {
			result.Storage.Name = fmt.Sprintf("%v", name)
		}

		id, ok := storagesystem["id"]
		if ok {
			result.Storage.Id = fmt.Sprintf("%v", id)
		}

		codelevel, ok := storagesystem["code_level"]
		if ok {
			result.Storage.Version = fmt.Sprintf("%v", codelevel)
		}

		var nodes restclient.Nodes
		nodes, err = s.fsrestclient.Lsnode()
		if err != nil {
			err = s.fsrestclient.Reconnect()
			if err != nil {
				klog.Errorf("fails to authenticate rest server, err:%s", err)
				return nil, err
			}
		}

		for _, node := range nodes {
			status, ok := node["status"]
			if !ok || "online" != status {
				result.Storage.State = pb_vlalpha.StorageState_StorageDegraded
				break
			}
		}
	}

	klog.Infof("End to get storage info, TimeDuration:%v", time.Since(bT))
	return result, nil
}
