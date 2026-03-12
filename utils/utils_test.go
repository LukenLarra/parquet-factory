// Copyright 2022 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/parquet-factory/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetHourOnly(t *testing.T) {
	inputDate := time.Date(2022, time.January, 1, 1, 2, 3, 0, time.UTC)
	wantDate := time.Date(2022, time.January, 1, 1, 0, 0, 0, time.UTC)
	gotDate := utils.GetHourOnly(inputDate)

	assert.Equal(t, wantDate, gotDate)
}

func TestGenerateParquetFilepath(t *testing.T) {
	var (
		timestamp = time.Date(2022, time.January, 1, 1, 2, 3, 0, time.UTC)
		prefix    = "prefix"
		filename  = "filename"
	)
	t.Run("no index", func(t *testing.T) {
		wantPath := "prefix/filename/hourly/date=2022-01-01/hour=01/filename.parquet"
		fmt.Println(wantPath)
		path := utils.GenerateParquetFilepath(timestamp, prefix, filename, -1)
		assert.Equal(t, wantPath, path)
	})

	t.Run("with index", func(t *testing.T) {
		var index = 1
		wantPath := "prefix/filename/hourly/date=2022-01-01/hour=01/filename-1.parquet"

		gotPath := utils.GenerateParquetFilepath(timestamp, prefix, filename, index)
		assert.Equal(t, wantPath, gotPath)
	})
}

func TestGenerateHourPrefix(t *testing.T) {
	var (
		timestamp = time.Date(2022, time.January, 1, 1, 2, 3, 0, time.UTC)
		prefix    = "prefix"
		filename  = "filename"
	)

	wantPrefix := "prefix/filename/hourly/date=2022-01-01/hour=01/"
	gotPrefix := utils.GenerateHourPrefix(timestamp, prefix, filename)

	assert.Equal(t, wantPrefix, gotPrefix)
}

func TestGetPathFromRawMsg(t *testing.T) {
	testFilesDir := "../testdata/kafka_messages/"

	e := filepath.WalkDir(testFilesDir, func(path string, d fs.DirEntry, err error) error {
		assert.NoError(t, err)
		fileInfo, err := os.Stat(path)
		assert.NoError(t, err)
		if fileInfo.IsDir() {
			return err
		}
		message, err := os.ReadFile(path) // #nosec G304 G122
		assert.NoError(t, err)
		archivePath, err := utils.GetPathFromRawMsg(message)
		assert.NoError(t, err)
		if strings.Contains(path, "empty") {
			assert.Empty(t, archivePath)
		} else {
			assert.NotEmpty(t, archivePath)
		}
		return err
	})
	assert.NoError(t, e)

	testJSON := []byte("{\"path\": \"archives/test/path\", \"garbage\":\"garbage\"}")
	archivePath, err := utils.GetPathFromRawMsg(testJSON)
	assert.NoError(t, err)
	assert.NotEmpty(t, archivePath)

	_, err = utils.GetPathFromRawMsg([]byte("make fail"))
	assert.Error(t, err)
}

func TestAdd(t *testing.T) {
	aps := utils.NewArchivePathSet()
	assert.True(t, aps.Add("ccx"))
	assert.False(t, aps.Add("ccx"))
	assert.Equal(t, aps.Size(), 1)
	assert.True(t, aps.Add("test"))
	assert.Equal(t, aps.Size(), 2)
}

func TestRemove(t *testing.T) {
	// setup
	aps := utils.NewArchivePathSet()
	assert.True(t, aps.Add("ccx"))
	assert.True(t, aps.Add("test"))
	assert.True(t, aps.Add("test0"))
	assert.True(t, aps.Add("test1"))
	assert.True(t, aps.Add("test2"))
	assert.Equal(t, aps.Size(), 5)

	// remove non existing element
	assert.False(t, aps.Remove("uhm"))
	assert.Equal(t, aps.Size(), 5)

	// remove 2 existing elements
	assert.True(t, aps.Remove("ccx"))
	assert.True(t, aps.Remove("test"))
	assert.Equal(t, aps.Size(), 3)
}

func TestContains(t *testing.T) {
	// setup
	aps := utils.NewArchivePathSet()
	assert.True(t, aps.Add("ccx"))
	assert.True(t, aps.Add("test"))
	assert.True(t, aps.Add("test0"))
	assert.True(t, aps.Add("test1"))
	assert.True(t, aps.Add("test2"))
	assert.Equal(t, aps.Size(), 5)

	assert.True(t, aps.Contains("ccx"))
	assert.False(t, aps.Contains("uhm"))
}

func TestRange(t *testing.T) {
	// setup
	aps := utils.NewArchivePathSet()
	assert.True(t, aps.Add("ccx"))
	assert.True(t, aps.Add("test"))
	assert.True(t, aps.Add("test0"))
	assert.True(t, aps.Add("test1"))
	assert.True(t, aps.Add("test2"))
	assert.Equal(t, aps.Size(), 5)

	testList := make([]string, 0, 5)
	assert.Len(t, testList, 0)
	for path := range aps.Range() {
		testList = append(testList, path)
	}
	assert.Len(t, testList, 5)

	// usage not recommended outside tests
	for path := range aps.Range() {
		aps.Remove(path)
	}
	assert.Equal(t, aps.Size(), 0)
}
