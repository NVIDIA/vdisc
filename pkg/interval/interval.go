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

package interval

type Interval struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

func Intersection(a, b Interval) (*Interval, bool) {
	start := maxI64(a.Start, b.Start)
	end := minI64(a.End, b.End)
	if start < end {
		return &Interval{start, end}, true
	}
	return nil, false
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
