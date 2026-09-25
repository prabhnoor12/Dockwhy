package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/prabhnoor12/dockwhy/internal/diagnosis"
)

// ExitCodeText writes exit code information as human-readable text.
func ExitCodeText(w io.Writer, info diagnosis.ExitCodeInfo) error {
	fmt.Fprintf(w, "Exit Code: %d (%s)\n", info.Code, info.Name)
	fmt.Fprintf(w, "Category:  %s\n", info.Category)
	fmt.Fprintf(w, "\n%s\n", info.Description)
	if info.Signal != "" {
		fmt.Fprintf(w, "\nSignal: %s\n", info.Signal)
	}
	if len(info.Causes) > 0 {
		fmt.Fprintln(w, "\nCommon causes:")
		for _, cause := range info.Causes {
			fmt.Fprintf(w, "  - %s\n", cause)
		}
	}
	if len(info.Fixes) > 0 {
		fmt.Fprintln(w, "\nRecommended fixes:")
		for _, fix := range info.Fixes {
			fmt.Fprintf(w, "  - %s\n", fix)
		}
	}
	return nil
}

// ExitCodeJSON writes exit code information as JSON.
func ExitCodeJSON(w io.Writer, info diagnosis.ExitCodeInfo) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}
