# Copyright © 2019 NVIDIA Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
@0xc6455bc28c9de795;

using Go = import "/std/go.capnp";
$Go.package("vdisc_types");
$Go.import("github.com/NVIDIA/vdisc/pkg/vdisc/types");

using V1 = import "/pkg/vdisc/types/v1/vdisc_v1.capnp";

struct VDisc {
  v1 @0 :V1.VDisc;
}
