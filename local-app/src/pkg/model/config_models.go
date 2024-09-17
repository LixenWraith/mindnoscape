// Package model defines the data structures used throughout the Mindnoscape application.
package model

type Config struct {
	DatabaseType        string `json:"database_type"`
	DatabaseDir         string `json:"database_dir"`
	DatabaseFile        string `json:"database_file"`
	LogFolder           string `json:"log_folder"`
	CommandLog          string `json:"command_log"`
	ErrorLog            string `json:"error_log"`
	InfoLog             string `json:"info_log"`
	DefaultUser         string `json:"default_user"`
	DefaultUserActive   bool   `json:"default_user_active"`
	DefaultUserPassword string `json:"default_user_password"`
}
