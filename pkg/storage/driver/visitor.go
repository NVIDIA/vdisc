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
package driver

import (
	"context"
	"os"
	"sync"
)

type Visitor interface {
	VisitDir(baseURL string, files []os.FileInfo) error
}

type VisitorTraversal interface {
	DepthFirst() bool
}

type VisitorConcurrency interface {
	Concurrency() int
}

type VisitorPredicate interface {
	ShouldVisitDir(url string) (bool, error)
}

func visitWorker(ctx context.Context, workerId int, drvr Readdirer, paths chan string, nextPaths chan string, errors chan error, visitor Visitor, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case dirPath, ok := <-paths:
			if !ok {
				return
			}

			if vp, ok := visitor.(VisitorPredicate); ok {
				var err error
				shouldVisit, err := vp.ShouldVisitDir(dirPath)
				if err != nil {
					errors <- err
					return
				}
				if !shouldVisit {
					continue
				}
			}

			files, err := drvr.Readdir(ctx, dirPath)
			if err != nil {
				errors <- err
				return
			}
			err = visitor.VisitDir(dirPath, files)
			if err != nil {
				errors <- err
				return
			}
			for _, f := range files {
				if f.IsDir() {
					fileName := f.Name()
					if fileName == "." || fileName == ".." {
						continue
					}
					if dirPath != "" {
						fileName = dirPath + "/" + fileName
					}
					select {
					case nextPaths <- fileName:
						break
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

func Visit(ctx context.Context, drvr Readdirer, root string, visitor Visitor) error {
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var maxWorkers int
	if vc, ok := visitor.(VisitorConcurrency); ok {
		maxWorkers = vc.Concurrency()
	}

	if maxWorkers <= 0 {
		maxWorkers = 32
	}

	nextPaths := []string{root}

	for len(nextPaths) > 0 {
		var paths []string

		var depthFirst bool
		if vt, ok := visitor.(VisitorTraversal); ok {
			depthFirst = vt.DepthFirst()
		}

		if depthFirst {
			// If in depth-first mode, we want to keep grabbing the top of the stack and process only that each iteration.
			amt := len(nextPaths) - minI(len(nextPaths), maxWorkers)
			paths = nextPaths[amt:]
			nextPaths = nextPaths[:amt]
		} else {
			paths = nextPaths
			nextPaths = make([]string, 0)
		}

		numWorkers := minI(len(paths), maxWorkers)

		inputPaths := make(chan string)
		resultPaths := make(chan string)
		errors := make(chan error, numWorkers)

		// Launch workers for this iteration.
		wg := sync.WaitGroup{}
		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go visitWorker(ctx, i, drvr, inputPaths, resultPaths, errors, visitor, &wg)
		}

		// Feed workers.
		go func() {
			for _, p := range paths {
				inputPaths <- p
			}
			// Signal there's no more work.
			close(inputPaths)
			wg.Wait()
			// Terminate the accumulator.
			close(resultPaths)
		}()

		// Accumulate results.
	accumulate:
		for {
			select {
			case p, ok := <-resultPaths:
				if !ok {
					break accumulate
				}
				nextPaths = append(nextPaths, p)
			case err := <-errors:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
