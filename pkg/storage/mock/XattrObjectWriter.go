// Code generated by mockery v1.0.0. DO NOT EDIT.

package mockdriver

import driver "github.com/NVIDIA/vdisc/pkg/storage/driver"
import mock "github.com/stretchr/testify/mock"

// XattrObjectWriter is an autogenerated mock type for the XattrObjectWriter type
type XattrObjectWriter struct {
	mock.Mock
}

// Abort provides a mock function with given fields:
func (_m *XattrObjectWriter) Abort() {
	_m.Called()
}

// Commit provides a mock function with given fields:
func (_m *XattrObjectWriter) Commit() (driver.CommitInfo, error) {
	ret := _m.Called()

	var r0 driver.CommitInfo
	if rf, ok := ret.Get(0).(func() driver.CommitInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(driver.CommitInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetXattr provides a mock function with given fields: name, value
func (_m *XattrObjectWriter) SetXattr(name string, value []byte) error {
	ret := _m.Called(name, value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []byte) error); ok {
		r0 = rf(name, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Write provides a mock function with given fields: p
func (_m *XattrObjectWriter) Write(p []byte) (int, error) {
	ret := _m.Called(p)

	var r0 int
	if rf, ok := ret.Get(0).(func([]byte) int); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
