@0xc6455bc28c9de795;

using Go = import "/std/go.capnp";
$Go.package("vdisc_types");
$Go.import("github.com/NVIDIA/vdisc/pkg/vdisc/types");

using V1 = import "/pkg/vdisc/types/v1/vdisc_v1.capnp";

struct VDisc {
  v1 @0 :V1.VDisc;
}
