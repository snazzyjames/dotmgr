package cli

import (
	"fmt"
	"os"
)

func step(format string, a ...any) {
	fmt.Println()
	fmt.Printf("▸ %s\n", fmt.Sprintf(format, a...))
}

func info(format string, a ...any) {
	fmt.Printf("  → %s\n", fmt.Sprintf(format, a...))
}

func errf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", fmt.Sprintf(format, a...))
}
