package animed

// A package which can be obtained
// from a backend via List() and Search()
// and downloaded via Get()
type Package struct {
	ID       uint64 `capid:"0"`
	Filename string `capid:"1"`
}

// document the interface to call a backend
type Backend interface {
	// outputs all packages the backend can supply
	// (this throws an error if the backend does not support enumeration)
	List() (packs []Package, err error)

	// Search for a query in the backend
	// returns packages which can be downloaded
	Search(query string) (packs []Package, err error)

	// Download a specified package
	Get(pack Package, destination string)

	// Query the Progress of a download
	Progress(pack Package) (progess float32, err error)

	// set a completion callback
	OnComplete(callback interface {
		Complete(Package)
	})
}
