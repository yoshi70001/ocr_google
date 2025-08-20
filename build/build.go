package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	platforms = map[string][]string{
		"windows": {"amd64"},
		"linux":   {"amd64"},
		"darwin":  {"amd64"},
	}
	appName = "googleDocsOCR"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run build/build.go <version>")
		os.Exit(1)
	}
	version := os.Args[1]

	releaseDir := "release"
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		fmt.Printf("Error creating release directory: %v\n", err)
		os.Exit(1)
	}

	for goos, archs := range platforms {
		for _, goarch := range archs {
			fmt.Printf("Building for %s/%s...\n", goos, goarch)
			outputName := fmt.Sprintf("%s-%s-%s", appName, goos, goarch)
			if goos == "windows" {
				outputName += ".exe"
			}

			cmd := exec.Command("go", "build", "-o", filepath.Join(releaseDir, outputName), "-ldflags", fmt.Sprintf("-X main.version=%s", version), ".")
			cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Printf("Error building for %s/%s: %v\n", goos, goarch, err)
			}
		}
	}

	fmt.Println("Build complete.")
}
