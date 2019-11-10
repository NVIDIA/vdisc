// Copyright © 2019 NVIDIA Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/NVIDIA/vdisc/pkg/storage/driver"
	"github.com/NVIDIA/vdisc/pkg/storage/mock"
)

func mockFile(name string, isDir bool) *mockdriver.FileInfo {
	mf := &mockdriver.FileInfo{}
	mf.On("Name").Return(name)
	mf.On("IsDir").Return(isDir)
	return mf
}

func TestVisit(t *testing.T) {
	ctx := mock.Anything
	drvr := &mockdriver.Readdirer{}
	drvr.On("Readdir", ctx, "").Return([]os.FileInfo{
		mockFile("A", true),
		mockFile("B", false),
	}, nil).Once()
	drvr.On("Readdir", ctx, "A").Return([]os.FileInfo{
		mockFile("AA", true),
		mockFile("AB", true),
		mockFile("AC", false),
	}, nil).Once()
	drvr.On("Readdir", ctx, "A/AA").Return([]os.FileInfo{
		mockFile("1", false),
		mockFile("2", false),
	}, nil).Once()
	drvr.On("Readdir", ctx, "A/AB").Return([]os.FileInfo{
		mockFile("3", false),
		mockFile("4", false),
		mockFile("5", false),
	}, nil).Once()
	visitor := &Visitor{}
	visitor.On("VisitDir", "", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 2)
	}).Return(nil).Once()
	visitor.On("VisitDir", "A", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 3)
	}).Return(nil).Once()
	visitor.On("VisitDir", "A/AA", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 2)
	}).Return(nil).Once()
	visitor.On("VisitDir", "A/AB", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 3)
	}).Return(nil).Once()
	visitor.On("DepthFirst").Return(false)
	assert.Nil(t, driver.Visit(context.Background(), drvr, "", visitor))
	drvr.AssertExpectations(t)
	visitor.AssertExpectations(t)
}

func TestVisitDepth(t *testing.T) {
	ctx := mock.Anything
	drvr := &mockdriver.Readdirer{}
	drvr.On("Readdir", ctx, "").Return([]os.FileInfo{
		mockFile("A", true),
		mockFile("B", true),
		mockFile("C", true),
	}, nil).Once()
	drvr.On("Readdir", ctx, "A").Return([]os.FileInfo{
		mockFile("AA", true),
	}, nil).Once()
	drvr.On("Readdir", ctx, "A/AA").Return([]os.FileInfo{
		mockFile("1", false),
		mockFile("2", false),
	}, nil).Once()
	drvr.On("Readdir", ctx, "B").Return([]os.FileInfo{
		mockFile("BB", true),
	}, nil).Once()
	drvr.On("Readdir", ctx, "C").Return([]os.FileInfo{
		mockFile("CC", true),
	}, nil).Once()
	drvr.On("Readdir", ctx, "B/BB").Return([]os.FileInfo{
		mockFile("3", false),
		mockFile("4", false),
	}, nil).Once()
	drvr.On("Readdir", ctx, "C/CC").Return([]os.FileInfo{
		mockFile("5", false),
		mockFile("6", false),
	}, nil).Once()
	visitor := &Visitor{}
	visitor.On("VisitDir", "", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 3)
	}).Return(nil).Once()
	visitor.On("VisitDir", "A", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 1)
	}).Return(nil).Once()
	visitor.On("VisitDir", "A/AA", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 2)
	}).Return(nil).Once()
	visitor.On("VisitDir", "B", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 1)
	}).Return(nil).Once()
	visitor.On("VisitDir", "B/BB", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 2)
	}).Return(nil).Once()
	visitor.On("VisitDir", "C", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 1)
	}).Return(nil).Once()
	visitor.On("VisitDir", "C/CC", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 2)
	}).Return(nil).Once()
	visitor.On("DepthFirst").Return(true)
	assert.Nil(t, driver.Visit(context.Background(), drvr, "", visitor))
	drvr.AssertExpectations(t)
	visitor.AssertExpectations(t)
}

func TestVisitError(t *testing.T) {
	ctx := mock.Anything
	drvr := &mockdriver.Readdirer{}
	drvr.On("Readdir", ctx, "").Return([]os.FileInfo{
		mockFile("A", true),
		mockFile("B", false),
	}, nil).Once()
	drvr.On("Readdir", ctx, "A").Return(nil, fmt.Errorf("expected error")).Once()
	visitor := &Visitor{}
	visitor.On("VisitDir", "", mock.Anything).Run(func(args mock.Arguments) {
		files := args.Get(1).([]os.FileInfo)
		assert.Len(t, files, 2)
	}).Return(nil).Once()
	visitor.On("DepthFirst").Return(false)
	assert.EqualError(t, driver.Visit(context.Background(), drvr, "", visitor), "expected error")
	drvr.AssertExpectations(t)
	visitor.AssertExpectations(t)
}

// Visitor is an autogenerated mock type for the Visitor type
type Visitor struct {
	mock.Mock
}

// VisitDir provides a mock function with given fields: baseURL, files
func (_m *Visitor) VisitDir(baseURL string, files []os.FileInfo) error {
	ret := _m.Called(baseURL, files)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []os.FileInfo) error); ok {
		r0 = rf(baseURL, files)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DepthFirst provides a mock function with given fields:
func (_m *Visitor) DepthFirst() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
