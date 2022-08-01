// Embed setup files

package setup

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed rpi
var Files embed.FS

// Export copies out server setup files for the specified server host.
func Export(forHost string, toPath string) error {

	// all embedded files for host
	ss, err := Files.ReadDir(forHost)
	if err != nil {
		return err
	}

	if len(ss) > 0 {
		// create setup directory
		err = os.MkdirAll(toPath, 0750)
		if err != nil {
			return err
		}
	}

	for _, s:= range ss {
		// read bytes
		sb, err := Files.ReadFile(s.Name())
		if err != nil {
			return err
		}

		// write file
		err = os.WriteFile(filepath.Join(toPath, s.Name()), sb, 0640)
		if err != nil {
			return err
		}
	}

	return nil
}
