package storage

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"mindnoscape/local-app/src/pkg/model"
	"os"
	"path/filepath"
)

// FileExport exports a mindmap to a file in the specified format (JSON or XML).
func FileExport(mindmap *model.Mindmap, filename string, format string) error {
	// Marshal the mindmap to the specified format
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.MarshalIndent(mindmap, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(mindmap, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal mindmap: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the data to the file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// FileImport imports a mindmap from a file in the specified format (JSON or XML).
func FileImport(filename string, format string) (*model.Mindmap, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the data into a mindmap structure
	var importedMindmap model.Mindmap
	switch format {
	case "json":
		err = json.Unmarshal(data, &importedMindmap)
	case "xml":
		err = xml.Unmarshal(data, &importedMindmap)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &importedMindmap, nil
}
