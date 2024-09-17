package storage

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"mindnoscape/local-app/src/pkg/log"
	"mindnoscape/local-app/src/pkg/model"
)

// FileExport exports a mindmap to a file in the specified format (JSON or XML).
func FileExport(mindmap *model.Mindmap, filename string, format string, logger *log.Logger) error {
	logger.Info(context.Background(), "Exporting mindmap to file", log.Fields{
		"mindmapID": mindmap.ID,
		"filename":  filename,
		"format":    format,
	})

	// Marshal the mindmap to the specified format
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.MarshalIndent(mindmap, "", "  ")
	case "xml":
		data, err = xml.MarshalIndent(mindmap, "", "  ")
	default:
		logger.Error(context.Background(), "Unsupported export format", log.Fields{"format": format})
		return fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		logger.Error(context.Background(), "Failed to marshal mindmap", log.Fields{"error": err, "format": format})
		return fmt.Errorf("failed to marshal mindmap: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error(context.Background(), "Failed to create directory", log.Fields{"error": err, "directory": dir})
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the data to the file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		logger.Error(context.Background(), "Failed to write file", log.Fields{"error": err, "filename": filename})
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info(context.Background(), "Mindmap exported successfully", log.Fields{
		"mindmapID": mindmap.ID,
		"filename":  filename,
		"format":    format,
	})
	return nil
}

// FileImport imports a mindmap from a file in the specified format (JSON or XML).
func FileImport(filename string, format string, logger *log.Logger) (*model.Mindmap, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		logger.Error(context.Background(), "Failed to read file", log.Fields{"error": err, "filename": filename})
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the data into a mindmap structures
	var importedMindmap model.Mindmap
	switch format {
	case "json":
		err = json.Unmarshal(data, &importedMindmap)
	case "xml":
		err = xml.Unmarshal(data, &importedMindmap)
	default:
		logger.Error(context.Background(), "Unsupported import format", log.Fields{"format": format})
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		logger.Error(context.Background(), "Failed to unmarshal data", log.Fields{"error": err, "format": format})
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	logger.Info(context.Background(), "Mindmap imported successfully", log.Fields{
		"filename":  filename,
		"format":    format,
		"mindmapID": importedMindmap.ID,
	})
	return &importedMindmap, nil
}
