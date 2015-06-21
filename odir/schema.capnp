@0xfd68025cd6b0a557;
using Go = import "go.capnp";

$Go.package("main");
$Go.import("testpkg");


struct PackageCapn { 
   iD        @0:   UInt64; 
   filename  @1:   Text; 
} 

##compile with:

##
##
##   capnp compile -ogo odir/schema.capnp

