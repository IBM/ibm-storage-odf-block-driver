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
package rest

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"time"

	"k8s.io/klog"
)

type Config struct {
	Host     string
	Username string
	Password string
}

type FSRestClient struct {
	Client     *http.Client
	RestConfig Config
	BaseURL    string
	token      *string // use nil as invalid token
}

func NewFSRestClient(config *Config) (*FSRestClient, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConnsPerHost: 1024,
	}

	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: tr,
	}

	c := &FSRestClient{
		Client:     client,
		BaseURL:    fmt.Sprintf("https://%s:7443/rest", config.Host),
		RestConfig: *config,
		token:      nil,
	}

	if err := c.authenticate(); err != nil {
		return nil, err
	}

	return c, nil
}

type authenResult map[string]interface{}

func (c *FSRestClient) authenticate() error {
	c.token = nil
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "auth"), nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Auth-Username", c.RestConfig.Username)
	req.Header.Set("X-Auth-Password", c.RestConfig.Password)

	req.Header.Set("Connection", "keep-alive")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var out authenResult
	if err = json.Unmarshal(body, &out); err != nil {
		return err
	}

	token, ok := out["token"]
	if !ok {
		return fmt.Errorf("token isn't included, %v", out)
	}

	tokenType := reflect.TypeOf(token).Kind()
	if reflect.String != tokenType {
		return fmt.Errorf("token type isn't string, %v, %s", token, tokenType)
	}

	tokenStr := token.(string)
	c.token = &tokenStr

	return nil
}

func (c *FSRestClient) Reconnect() error {
	return c.authenticate()
}

func (c *FSRestClient) Do(req *http.Request) ([]byte, error) {

	if req == nil {
		return nil, fmt.Errorf("invalid parameter, abort")
	}

	if c.token == nil {
		if err := c.authenticate(); err != nil {
			return nil, err
		}
	}

	req.Header.Set("X-Auth-Token", *c.token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

type StorageSystem map[string]interface{}

func (c *FSRestClient) Lssystem() (StorageSystem, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "lssystem"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)

	if err != nil {
		klog.Errorf("body %s", body)
		klog.Errorf("Lssystem err %v", err)
		return nil, err
	}

	var storagesystem StorageSystem
	if err = json.Unmarshal(body, &storagesystem); err != nil {
		klog.Errorf("Lssystem err %v, body %s", err, body)
		return nil, err
	}

	return storagesystem, nil
}

type Nodes []map[string]string

func (c *FSRestClient) Lsnode() (Nodes, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "lsnode"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)

	if err != nil {
		klog.Errorf("body %s", body)
		klog.Errorf("Lsnode err %v", err)
		return nil, err
	}

	var nodes Nodes
	if err = json.Unmarshal(body, &nodes); err != nil {
		klog.Errorf("Lsnode err %v, body %s", err, body)
		return nil, err
	}

	return nodes, nil
}

type SystemStats []map[string]string

func (c *FSRestClient) Lssystemstats() (SystemStats, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "lssystemstats"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		klog.Errorf("body %s", body)
		return nil, err
	}

	//fmt.Print(req, body)

	var stats SystemStats
	if err = json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

type Users []map[string]interface{}

func (c *FSRestClient) Lsuser() (Users, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "lsuser"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)

	if err != nil {
		klog.Errorf("body %s", body)
		klog.Errorf("Lsuser err %v", err)
		return nil, err
	}

	var users Users
	if err = json.Unmarshal(body, &users); err != nil {
		klog.Errorf("Lsuser err %v, body %s", err, body)
		return nil, err
	}

	return users, nil
}

// List of vdisk, result of lsvdisk
type VdiskList []map[string]interface{}

func (c *FSRestClient) Lsvdisk() (VdiskList, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "lsvdisk"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		klog.Errorf("body %s", body)
		return nil, err
	}

	var stats VdiskList
	if err = json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// vdisk detail info, result of lsvdisk <vdisk id>
type VdiskInfo []map[string]interface{}

func (c *FSRestClient) LsvdiskInfo(vdiskId string) (VdiskInfo, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s/%s", c.BaseURL, "lsvdisk", vdiskId), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		klog.Errorf("body %s", body)
		return nil, err
	}

	var stats VdiskInfo
	if err = json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// Pool list, result of lsmdisk
type PoolList []map[string]interface{}

func (c *FSRestClient) Lsmdisk() (PoolList, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "lsmdisk"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		klog.Errorf("body %s", body)
		return nil, err
	}

	var stats PoolList
	if err = json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// Pool info, result of lsmdiskgrp <pool id>
type PoolInfo map[string]interface{}

func (c *FSRestClient) LsmdiskInfo(mdiskId int) (PoolInfo, error) {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s/%d", c.BaseURL, "lsmdiskgrp", mdiskId), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		klog.Errorf("body %s", body)
		return nil, err
	}

	var stats PoolInfo
	if err = json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

