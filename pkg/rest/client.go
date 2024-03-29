/**
 * Copyright contributors to the ibm-storage-odf-block-driver project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rest

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"time"

	drivermanager "github.com/IBM/ibm-storage-odf-block-driver/pkg/driver"

	corev1 "k8s.io/api/core/v1"
	log "k8s.io/klog"
)

type Config struct {
	Host     string
	Username string
	Password string
}

const (
	FailedEventThreshold = time.Minute * 2 // 2 minutes
)

type FSRestClient struct {
	Client        *http.Client
	RestConfig    Config
	BaseURL       string
	token         *string // use nil as invalid token
	DriverManager *drivermanager.DriverManager
	PostRequester *Requester

	failedTime time.Time
	bNotified  bool
}

// For easy mock the request response
type Poster func(req *http.Request, c *FSRestClient) ([]byte, int, error)

type Requester struct {
	poster Poster
}

func NewRequester(p Poster) *Requester {
	return &Requester{poster: p}
}

func (c *FSRestClient) NewFSRestClient(config Config, driverManager *drivermanager.DriverManager) (*FSRestClient, error) {
	tr := &http.Transport{
		// #nosec
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

	cl := &FSRestClient{
		Client:        client,
		BaseURL:       fmt.Sprintf("https://%s:7443/rest", config.Host),
		RestConfig:    config,
		token:         nil,
		DriverManager: driverManager,
		PostRequester: NewRequester(doRequest),
	}

	if err := cl.authenticate(); err != nil {
		return nil, err
	}

	return cl, nil
}

type authenResult map[string]interface{}

func (c *FSRestClient) authenticate() error {
	if !c.bNotified && !c.failedTime.Equal(time.Time{}) && time.Since(c.failedTime) > FailedEventThreshold {
		mgr := c.DriverManager
		if mgr != nil {
			if err := mgr.SendK8sEvent(corev1.EventTypeWarning, drivermanager.AuthFailure, drivermanager.AuthFailureMessage); err == nil {
				c.bNotified = true
			}
		}
	}

	c.token = nil
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.BaseURL, "auth"), nil)
	if err != nil {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		return err
	}

	req.Header.Set("X-Auth-Username", c.RestConfig.Username)
	req.Header.Set("X-Auth-Password", c.RestConfig.Password)

	req.Header.Set("Connection", "keep-alive")

	resp, err := c.Client.Do(req)
	if err != nil {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		return err
	}

	if resp.StatusCode != 200 {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		errMsg := fmt.Sprintf("Authentication failed with response code: %d", resp.StatusCode)
		return errors.New(errMsg)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		return err
	}

	var out authenResult
	if err = json.Unmarshal(body, &out); err != nil {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		return err
	}

	token, ok := out["token"]
	if !ok {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		return fmt.Errorf("token isn't included, %v", out)
	}

	tokenType := reflect.TypeOf(token).Kind()
	if reflect.String != tokenType {
		if c.failedTime.Equal(time.Time{}) {
			c.failedTime = time.Now()
		}
		return fmt.Errorf("token type isn't string, %v, %s", token, tokenType)
	}

	tokenStr := token.(string)
	c.token = &tokenStr

	if c.bNotified {
		mgr := c.DriverManager
		if mgr != nil {
			if err = mgr.SendK8sEvent(corev1.EventTypeNormal, drivermanager.AuthSuccess, drivermanager.AuthSuccessMessage); err == nil {
				c.bNotified = false
			}
		}
	}
	c.failedTime = time.Time{}

	return nil
}

func (c *FSRestClient) retryDo(url string, jsonStr string) ([]byte, error) {
	var reqBody io.Reader = nil
	if len(jsonStr) > 0 {
		reqBody = bytes.NewBufferString(jsonStr)
	}
	req, err := http.NewRequest("POST", url, reqBody)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		log.Errorf("Create request error for url: %s", url)
		return nil, err
	}
	body, statusCode, err := c.PostRequester.poster(req, c)

	retryCnt := 2
	for i := 0; i < retryCnt-1; i++ {
		if len(body) > 0 && statusCode >= http.StatusOK && statusCode < http.StatusBadRequest {
			return body, err
		}

		// Sometimes got the 'Invalid token error'.
		// Set the token to nil to do reauthentication
		c.token = nil
		body, statusCode, err = c.PostRequester.poster(req, c)
	}

	if statusCode >= http.StatusBadRequest {
		log.Errorf("Http request path %s response code is: %d after retry %d times", req.URL.Path, statusCode, retryCnt)
		if err == nil {
			err = errors.New("POST Request " + req.URL.Path + " error.")
		}
	}

	return body, err
}

func doRequest(req *http.Request, c *FSRestClient) ([]byte, int, error) {
	if req == nil {
		return nil, http.StatusBadRequest, errors.New("invalid parameter, abort")
	}

	if c.token == nil {
		if err := c.authenticate(); err != nil {
			log.Errorf("fails to authenticate rest server, err:%v", err)
			return nil, http.StatusUnauthorized, err
		}
	}

	req.Header.Set("X-Auth-Token", *c.token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

type StorageSystem map[string]interface{}

func (c *FSRestClient) Lssystem() (StorageSystem, error) {
	jsonStr := `{"gui":true,"bytes":true}`
	body, err := c.retryDo(fmt.Sprintf("%s/%s", c.BaseURL, "lssystem"), jsonStr)
	if err != nil {
		return nil, err
	}

	var storagesystem StorageSystem
	if err = json.Unmarshal(body, &storagesystem); err != nil {
		log.Errorf("Lssystem err %v, body %s", err, body)
		return nil, err
	}

	return storagesystem, nil
}

type Nodes []map[string]string

func (c *FSRestClient) Lsnode() (Nodes, error) {
	body, err := c.retryDo(fmt.Sprintf("%s/%s", c.BaseURL, "lsnode"), "")
	if err != nil {
		return nil, err
	}

	var nodes Nodes
	if err = json.Unmarshal(body, &nodes); err != nil {
		log.Errorf("Lsnode err %v, body %s", err, body)
		return nil, err
	}

	return nodes, nil
}

type SystemStats []map[string]string

func (c *FSRestClient) Lssystemstats() (SystemStats, error) {
	body, err := c.retryDo(fmt.Sprintf("%s/%s", c.BaseURL, "lssystemstats"), "")
	if err != nil {
		return nil, err
	}

	var stats SystemStats
	if err = json.Unmarshal(body, &stats); err != nil {
		log.Errorf("lssystemstats err %v, body %s", err, body)
		return nil, err
	}

	return stats, nil
}

type Users []map[string]interface{}

func (c *FSRestClient) Lscurrentuser() (Users, error) {
	body, err := c.retryDo(fmt.Sprintf("%s/%s", c.BaseURL, "lscurrentuser"), "")
	if err != nil {
		return nil, err
	}

	var users Users
	if err = json.Unmarshal(body, &users); err != nil {
		log.Errorf("Lscurrentuser err %v, body %s", err, body)
		return nil, err
	}

	return users, nil
}

// Pool list, result of lsmdiskgrp
type PoolList []map[string]interface{}

func (c *FSRestClient) Lsmdiskgrp() (PoolList, error) {
	jsonStr := `{"gui":true,"bytes":true}`
	body, err := c.retryDo(fmt.Sprintf("%s/%s", c.BaseURL, "lsmdiskgrp"), jsonStr)
	if err != nil {
		return nil, err
	}

	var stats PoolList
	if err = json.Unmarshal(body, &stats); err != nil {
		log.Errorf("Lsmdiskgrp err %v, body %s", err, body)
		return nil, err
	}

	return stats, nil
}

type MDisksList []map[string]interface{}

func (c *FSRestClient) LsAllMDisk() (MDisksList, error) {
	jsonStr := `{"gui":true,"bytes":true}`
	body, err := c.retryDo(fmt.Sprintf("%s/%s", c.BaseURL, "lsmdisk"), jsonStr)
	if err != nil {
		return nil, err
	}

	var stats MDisksList
	if err = json.Unmarshal(body, &stats); err != nil {
		log.Errorf("Lsmdisk for list err %v, body %s", err, body)
		return nil, err
	}

	return stats, nil
}

type SingleMDiskInfo map[string]interface{}

func (c *FSRestClient) LsSingleMDisk(diskID int) (SingleMDiskInfo, error) {
	jsonStr := `{"gui":true,"bytes":true}`
	body, err := c.retryDo(fmt.Sprintf("%s/%s/%d", c.BaseURL, "lsmdisk", diskID), jsonStr)
	if err != nil {
		return nil, err
	}

	var stats SingleMDiskInfo
	if err = json.Unmarshal(body, &stats); err != nil {
		log.Errorf("Lsmdisk for single disk err %v, body %s", err, body)
		return nil, err
	}

	return stats, nil
}

func (c *FSRestClient) UpdateCredentials(newConfig Config) error {
	if !reflect.DeepEqual(newConfig, c.RestConfig) {
		c.RestConfig = newConfig
		if err := c.authenticate(); err != nil {
			log.Errorf("Failed to authenticate rest server, err:%v", err)
			return err
		}
	}
	return nil

}
