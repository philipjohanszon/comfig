package comfig

import "os"

func Environment() string {
	if v := os.Getenv("env"); v != "" {
		return v
	}
	if v := os.Getenv("ENV"); v != "" {
		return v
	}
	return "local"
}
