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

package blockdev_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tnarg/go-tcmu"

	"github.com/NVIDIA/vdisc/pkg/blockdev"
)

func TestCmdPool0(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)
	err := pool.Close()
	assert.Nil(t, err)
}

type fakeHandler struct {
	ok    chan tcmu.SCSIResponse
	error chan error
}

func newFakeHandler() *fakeHandler {
	return &fakeHandler{
		ok:    make(chan tcmu.SCSIResponse),
		error: make(chan error),
	}
}

func (h *fakeHandler) HandleCommand(cmd *tcmu.SCSICmd) (rsp tcmu.SCSIResponse, err error) {
	select {
	case rsp = <-h.ok:
	case err = <-h.error:
	}
	return
}

type slowHandler struct{}

func newSlowHandler() *slowHandler {
	return &slowHandler{}
}

func (h *slowHandler) HandleCommand(cmd *tcmu.SCSICmd) (rsp tcmu.SCSIResponse, err error) {
	time.Sleep(1 * time.Second)
	rsp = cmd.Ok()
	return
}

func TestCmdPool1(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)

	h := newFakeHandler()

	in := make(chan *tcmu.SCSICmd)
	out := make(chan tcmu.SCSIResponse)

	err := pool.DevReady(h)(in, out)
	assert.Nil(t, err)

	err = pool.Close()
	assert.Nil(t, err)
}

func TestCmdPool2(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)

	h := newFakeHandler()

	in := make(chan *tcmu.SCSICmd)
	out := make(chan tcmu.SCSIResponse)

	err := pool.DevReady(h)(in, out)
	assert.Nil(t, err)

	close(in)

	select {
	case <-time.After(5 * time.Second):
		panic("never")
	case <-out:
	}

	err = pool.Close()
	assert.Nil(t, err)
}

func TestCmdPool3(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)

	h := newFakeHandler()

	in := make(chan *tcmu.SCSICmd)
	out := make(chan tcmu.SCSIResponse)

	err := pool.DevReady(h)(in, out)
	assert.Nil(t, err)

	req := &tcmu.SCSICmd{}
	expected := req.RespondStatus(7)

	select {
	case in <- req:
	case <-time.After(5 * time.Second):
		panic("never")
	}

	select {
	case h.ok <- req.RespondStatus(7):
	case <-time.After(5 * time.Second):
		panic("never")
	}

	var actual tcmu.SCSIResponse
	select {
	case actual = <-out:
	case <-time.After(5 * time.Second):
		panic("never")
	}

	assert.Equal(t, expected, actual, "response")

	close(in)
	select {
	case <-time.After(5 * time.Second):
		panic("never")
	case <-out:
	}

	err = pool.Close()
	assert.Nil(t, err)
}

func TestCmdPool4(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)

	h := newSlowHandler()

	in := make(chan *tcmu.SCSICmd)
	out := make(chan tcmu.SCSIResponse)

	err := pool.DevReady(h)(in, out)
	assert.Nil(t, err)

	go func() {
		<-out
	}()

	select {
	case in <- &tcmu.SCSICmd{}:
	case <-time.After(5 * time.Second):
		panic("never")
	}

	time.Sleep(10 * time.Millisecond)

	err = pool.Close()
	assert.Nil(t, err)

	done := make(chan interface{})
	close(done)

	select {
	case <-out:
		panic("never")
	case <-done:
	}

	close(in)
	select {
	case <-time.After(3 * time.Second):
	case <-out:
		panic("never")
	}

	close(out)
}

func TestCmdPool5(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)

	h := newFakeHandler()

	in := make(chan *tcmu.SCSICmd)
	out := make(chan tcmu.SCSIResponse)

	err := pool.DevReady(h)(in, out)
	assert.Nil(t, err)

	for i := byte(0); i < 255; i++ {
		req := &tcmu.SCSICmd{}
		expected := req.RespondStatus(i)

		select {
		case in <- req:
		case <-time.After(5 * time.Second):
			panic("never")
		}

		select {
		case h.ok <- req.RespondStatus(i):
		case <-time.After(5 * time.Second):
			panic("never")
		}

		var actual tcmu.SCSIResponse
		select {
		case actual = <-out:
		case <-time.After(5 * time.Second):
			panic("never")
		}

		assert.Equal(t, expected, actual, "response")
	}

	close(in)
	select {
	case <-time.After(5 * time.Second):
		panic("never")
	case <-out:
	}

	err = pool.Close()
	assert.Nil(t, err)
}

func TestCmdPool6(t *testing.T) {
	pool := blockdev.NewCmdPool(20, 1024)

	wg := &sync.WaitGroup{}
	for j := 0; j < 100; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			h := newFakeHandler()
			in := make(chan *tcmu.SCSICmd)
			out := make(chan tcmu.SCSIResponse)

			err := pool.DevReady(h)(in, out)
			assert.Nil(t, err)

			for i := byte(0); i < 255; i++ {
				req := &tcmu.SCSICmd{}
				expected := req.RespondStatus(i)

				select {
				case in <- req:
				case <-time.After(5 * time.Second):
					panic("never")
				}

				select {
				case h.ok <- req.RespondStatus(i):
				case <-time.After(5 * time.Second):
					panic("never")
				}

				var actual tcmu.SCSIResponse
				select {
				case actual = <-out:
				case <-time.After(5 * time.Second):
					panic("never")
				}
				assert.Equal(t, expected, actual, "response")
			}

			close(in)
			select {
			case <-time.After(5 * time.Second):
				panic("never")
			case <-out:
			}
		}()
	}

	wg.Wait()

	err := pool.Close()
	assert.Nil(t, err)
}
