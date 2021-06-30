package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/CanalTP/gormungandr/internal/coverage"
)

func GetFileWithFS(uri url.URL) ([]*coverage.Coverage, error) {
	//Get all files in directory params
	fileInfo, err := ioutil.ReadDir(uri.Path)
	if err != nil {
		return nil, err
	}

	coverages := make([]*coverage.Coverage, 0)
	for _, file := range fileInfo {
		f, err := os.Open(fmt.Sprintf("%s/%s", uri.Path, file.Name()))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var buffer bytes.Buffer
		if _, err = buffer.ReadFrom(f); err != nil {
			return nil, err
		}
		jsonData, err := ioutil.ReadAll(&buffer)
		if err != nil {
			return nil, err
		}
		coverage := &coverage.Coverage{}
		err = json.Unmarshal([]byte(jsonData), coverage)
		if err != nil {
			return nil, err
		}
		coverages = append(coverages, coverage)
	}
	return coverages, nil
}
