/*
 * Copyright 2026 Roberto Leinardi.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package config reads monmux's configuration file. There is very little of it
// on purpose: a serial to pin to, and where the two external tools live.
//
// Nothing here can widen what monmux is willing to do. The catalog decides which
// monitors and which inputs are write-enabled, and it is generated into the binary
// from a file read only at development time; the backend is chosen by the
// operating system and cannot be overridden. A configuration
// file can narrow a request or point at a different binary, and that is all -
// which is why an unknown key is an error rather than something to ignore.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// Directory is monmux's directory inside the configuration home.
	Directory = "monmux"
	// FileName is the configuration file itself.
	FileName = "config.yaml"

	// XDGConfigHome is the environment variable that relocates the
	// configuration home.
	XDGConfigHome = "XDG_CONFIG_HOME"
	// defaultHome is where the configuration home is when it is not relocated.
	defaultHome = ".config"
)

// ErrUnreadable reports a configuration file that exists but cannot be used.
var ErrUnreadable = errors.New("config: the configuration file cannot be read")

// Config is the whole of monmux's configuration.
type Config struct {
	// Serial pins every switch to one physical unit. It matches the
	// alphanumeric serial string only, exactly as --serial does.
	Serial string `yaml:"serial"`
	// DDCUtilPath is an absolute path to the ddcutil binary, for a Linux
	// system where it is not on PATH or where a specific build must be used.
	//
	//nolint:tagliatelle // the file is snake_case: these keys name the tools ddcutil and m1ddc
	DDCUtilPath string `yaml:"ddcutil_path"`
	// M1DDCPath is the same for m1ddc on macOS.
	//
	//nolint:tagliatelle // see above
	M1DDCPath string `yaml:"m1ddc_path"`
}

// Path returns where the configuration file is looked for:
// $XDG_CONFIG_HOME/monmux/config.yaml, or ~/.config/monmux/config.yaml when
// that variable is not set. The location is the same on Linux and macOS,
// because a user with both machines should not have to remember two.
func Path() (string, error) {
	home := os.Getenv(XDGConfigHome)

	if home == "" {
		user, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locating the home directory: %w", err)
		}

		home = filepath.Join(user, defaultHome)
	}

	return filepath.Join(home, Directory, FileName), nil
}

// Load reads the configuration from its usual location.
//
// No configuration file is the normal case, not an error: monmux works without
// one. A file that exists but cannot be parsed is an error, because the user
// wrote it meaning something by it.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	return LoadFrom(path)
}

// LoadFrom reads the configuration from a named file.
func LoadFrom(path string) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}

		return Config{}, fmt.Errorf("%w: %s: %w", ErrUnreadable, path, err)
	}

	config, err := Parse(contents)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}

	return config, nil
}

// Parse reads a configuration document.
//
// An unrecognized key is an error. A typo in a key that was silently ignored
// would leave the user believing a pin or a tool path is in effect when it is
// not, and for a serial pin that means writing to a monitor they meant to
// exclude.
func Parse(document []byte) (Config, error) {
	var config Config

	decoder := yaml.NewDecoder(bytes.NewReader(document))
	decoder.KnownFields(true)

	err := decoder.Decode(&config)
	if err != nil {
		// An empty file decodes to nothing at all, which is the same as having
		// no file.
		if errors.Is(err, io.EOF) {
			return Config{}, nil
		}

		return Config{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return config, nil
}
