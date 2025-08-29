package config

import "os"

func osHostnameReal() (string, error) { return os.Hostname() }
