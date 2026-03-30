package pklloader

import (
	"io/fs"
	"log"

	"github.com/apple/pkl-go/pkl"
)

// RunExternalReader starts a Pkl external module reader that serves files from
// the provided fs.FS under the given scheme. It communicates with the Pkl
// evaluator over stdin/stdout using the external reader protocol.
//
// This is intended to be called from a small main() in a cmd/ directory that
// the PklProject references as an external module reader:
//
//	func main() {
//	    pklloader.RunExternalReader("myscheme", schema.FS)
//	}
//
// The PklProject evaluatorSettings would then reference this binary:
//
//	evaluatorSettings {
//	  externalModuleReaders {
//	    ["myscheme"] {
//	      executable = "go"
//	      arguments { "run" "./cmd/pkl-reader" }
//	    }
//	  }
//	}
//
// Config files can then import modules via myscheme:/path/to/File.pkl.
func RunExternalReader(scheme string, fsys fs.FS) {
	client, err := pkl.NewExternalReaderClient(
		pkl.WithExternalClientFs(fsys, scheme),
	)
	if err != nil {
		log.Fatalf("pkl-loader: create external reader client: %v", err)
	}
	if err := client.Run(); err != nil {
		log.Fatalf("pkl-loader: external reader: %v", err)
	}
}

// RunExternalReaderMulti starts a Pkl external reader that serves multiple
// fs.FS sources, each under its own scheme. This allows a single binary to
// handle multiple custom schemes.
func RunExternalReaderMulti(schemes map[string]fs.FS) {
	opts := make([]func(*pkl.ExternalReaderClientOptions), 0, len(schemes))
	for scheme, fsys := range schemes {
		opts = append(opts, pkl.WithExternalClientFs(fsys, scheme))
	}
	client, err := pkl.NewExternalReaderClient(opts...)
	if err != nil {
		log.Fatalf("pkl-loader: create external reader client: %v", err)
	}
	if err := client.Run(); err != nil {
		log.Fatalf("pkl-loader: external reader: %v", err)
	}
}
