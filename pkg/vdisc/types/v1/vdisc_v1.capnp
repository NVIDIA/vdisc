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
@0xad3f2ae443d613d9;

using Go = import "/std/go.capnp";
$Go.package("vdisc_types_v1");
$Go.import("github.com/NVIDIA/vdisc/pkg/vdisc/types/v1");

#
# A read-only disc image comprised of a sequence of extents, each
# backed by a separate object.
#
struct VDisc {
  # The block size of every extent
  blockSize @0 :UInt16;

  # The file system type stored in this disc image
  fsType    @1 :Text;

  # A compressed representation of object URI prefixes
  uris      @2 :List(ITrie);

  # The extents that constitute this disc image
  extents   @3 :List(Extent);
}

#
# The ITrie is an inverted trie data structure for compressing object
# URIs.
#
struct ITrie {
    # The index of the parent of this node
    # The root node points to itself
    parent  @0 :UInt32;

    # The content of this segment of a URI
    # The root node content may be empty
    content @1 :Text;
}

#
# A disc image extent backed by an object.
#
struct Extent {
  # index into the "uris" inverted trie
  uriPrefix @0 :UInt32;

  # the last several characters of the object URI
  uriSuffix @1 :Text;

  # how many blocks of the disc image this extent consumes
  blocks    @2 :UInt32;

  # padding bytes in the final block
  padding   @3 :UInt16;
}
