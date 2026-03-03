// Copyright 2021 Red Hat, Inc
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

package s3writer_test

import (
	"context"
	"errors"
	"testing"

	s3mocks "github.com/RedHatInsights/insights-operator-utils/s3/mocks"
	"github.com/stretchr/testify/assert"
)

type testTableSchema struct {
	ID          string `parquet:"name=id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	CollectedAt int64  `parquet:"name=collected_at, type=INT64, logicaltype=TIMESTAMP, logicaltype.isadjustedtoutc=true, logicaltype.unit=MILLIS"`
}

var testRow = testTableSchema{
	ID:          "5d5892d3-1f74-4ccf-xxxx-548dfc9767aa",
	CollectedAt: 1612269,
}

func TestNewFile(t *testing.T) {
	mockClient := s3mocks.MockS3Client{}
	s3Writer := newMockS3Writer(t, &mockClient)
	t.Run("shouldn't return an error if the schema is valid", func(t *testing.T) {
		_, err := s3Writer.NewFile(context.TODO(), "my_file", &testTableSchema{})
		assert.NoError(t, err)
	})

	t.Run("should return an error if the schema is not a pointer", func(t *testing.T) {
		mockClient.Err = errors.New("an error")
		_, err := s3Writer.NewFile(context.TODO(), "my_file", testTableSchema{})
		assert.Error(t, err)
	})
}

func TestAddRow(t *testing.T) {
	mockClient := s3mocks.MockS3Client{}
	s3Writer := newMockS3Writer(t, &mockClient)

	s3file, err := s3Writer.NewFile(context.TODO(), "my_file", &testTableSchema{})
	assert.NoError(t, err)
	err = s3file.AddRow(testRow)
	assert.NoError(t, err)
}

func TestCloseFile(t *testing.T) {
	mockClient := s3mocks.MockS3Client{}
	s3Writer := newMockS3Writer(t, &mockClient)

	s3file, err := s3Writer.NewFile(context.TODO(), "my_file", &testTableSchema{})
	assert.NoError(t, err)
	err = s3file.AddRow(testRow)
	assert.NoError(t, err)
	// TODO: Add more methods to s3mocks.MockS3Client so that we can test the CloseFile function
	//err = s3file.CloseFile()
	// assert.NoError(t, err)
}
