package export

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nad/pkgview/internal/model"
)

var marshalJSON = JSON

func TXT(packages []model.Package) []byte {
	if len(packages) == 0 {
		return nil
	}
	lines := make([]string, 0, len(packages))
	for _, pkg := range packages {
		line := pkg.Name
		if pkg.Version != "" {
			line += "\t" + pkg.Version
		}
		line += "\t" + string(pkg.Source)
		lines = append(lines, line)
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func JSON(packages []model.Package) ([]byte, error) {
	return json.MarshalIndent(packages, "", "  ")
}

func WriteTXT(path string, packages []model.Package) error {
	return os.WriteFile(path, TXT(packages), 0o644)
}

func WriteJSON(path string, packages []model.Package) error {
	data, err := marshalJSON(packages)
	if err != nil {
		return fmt.Errorf("marshal export json: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
