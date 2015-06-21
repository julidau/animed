package main

import (
  capn "github.com/glycerine/go-capnproto"
  "io"
)




func (s *Package) Save(w io.Writer) error {
  	seg := capn.NewBuffer(nil)
  	PackageGoToCapn(seg, s)
    _, err := seg.WriteTo(w)
    return err
}
 


func (s *Package) Load(r io.Reader) error {
  	capMsg, err := capn.ReadFromStream(r, nil)
  	if err != nil {
  		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
        return err
  	}
  	z := ReadRootPackageCapn(capMsg)
      PackageCapnToGo(z, s)
   return nil
}



func PackageCapnToGo(src PackageCapn, dest *Package) *Package {
  if dest == nil {
    dest = &Package{}
  }
  dest.ID = src.ID()
  dest.Filename = src.Filename()

  return dest
}



func PackageGoToCapn(seg *capn.Segment, src *Package) PackageCapn {
  dest := AutoNewPackageCapn(seg)
  dest.SetID(src.ID)
  dest.SetFilename(src.Filename)

  return dest
}
