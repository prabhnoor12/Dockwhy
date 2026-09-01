package output

import (
	"encoding/json"
	"io"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

func JSON(w io.Writer, result diagnosis.Result) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
