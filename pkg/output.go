package pkg

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tools"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
)

// EmitFile writes the file with the provided contents in the output
// directory outDir.
func EmitFile(outDir, relPath string, contents []byte) error {
	if contents == nil {
		return nil
	}

	// we only want to write a file if there are contents to write
	p := path.Join(outDir, relPath)
	if err := tools.EnsureDir(path.Dir(p)); err != nil {
		return errors.Wrap(err, "creating directory")
	}

	f, err := os.Create(p)
	if err != nil {
		return errors.Wrap(err, "creating file")
	}
	defer contract.IgnoreClose(f)

	_, err = f.Write(contents)
	return err
}
