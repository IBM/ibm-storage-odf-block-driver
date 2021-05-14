package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	PoolConfigmapName      = "ibm-flashsystem-pools"
	PoolConfigmapMountPath = "/config"
	PoolConfigmapKey       = "pools"
)

type ScPoolMap struct {
	ScPool map[string]string `json:"storageclass_pool,omitempty"`
}

func GeneratePoolConfigmapContent(sp ScPoolMap) (string, error) {

	data, err := json.Marshal(sp)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func readPoolConfigMapFile() ([]byte, error) {
	poolPath := filepath.Join(PoolConfigmapMountPath, PoolConfigmapKey)

	file, err := os.Open(poolPath) // For read access.
	if err != nil {
		//		if os.IsNotExist(err) {
		//			return "", nil
		//		} else {
		return nil, err
		//		}
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func GetPoolConfigmapContent() (ScPoolMap, error) {
	var sp ScPoolMap

	content, err := readPoolConfigMapFile()
	if err != nil {
		return sp, err
	}

	err = json.Unmarshal(content, &sp)
	return sp, err
}
