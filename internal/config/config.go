package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frozzare/max/internal/cache"
	"github.com/frozzare/max/internal/task"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

var (
	// ErrUnmarshal is returned when config can't be unmarshaled.
	ErrUnmarshal = errors.New("max: can't unmarshal config value")
	// ErrCreateCache is returned when cache can't be created.
	ErrCreateCache = errors.New("max: can't create cache")
)

// Config represents a config file.
type Config struct {
	cache     *cache.Cache
	Args      map[string]interface{}
	Tasks     map[string]*task.Task
	Variables map[string]string
	Version   string
}

type base struct {
	Args      map[string]interface{}
	Tasks     map[string]interface{}
	Quiet     bool
	Variables map[string]string
	Version   string
}

// CreateCache creates a new cache.
func CreateCache() (*cache.Cache, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	return cache.New(filepath.Join(dir, ".max"))
}

// Default set default values to config struct.
func (c *Config) Default() {
	if cache, err := CreateCache(); err == nil {
		c.cache = cache
	}
}

// UnmarshalYAML implements yaml packages interface to unmarshal custom values.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var b *base

	if err := unmarshal(&b); err == nil {
		c.Args = b.Args
		c.Tasks = make(map[string]*task.Task)
		c.Variables = b.Variables
		c.Version = b.Version

		if c.Variables == nil {
			c.Variables = make(map[string]string)
		}

		// Loop over tasks to include and convert existing maps to tasks.
		for k, v := range b.Tasks {
			switch r := v.(type) {
			case string:
				if strings.Contains(r, "http") {
					t, err := includeHTTPTask(r, c.cache)
					if err != nil {
						return ErrUnmarshal
					}

					c.Tasks[k] = t
				} else if content, err := ioutil.ReadFile(r); err == nil {
					var t *task.Task
					if err := yaml.Unmarshal([]byte(content), &t); err == nil {
						c.Tasks[k] = t
					} else {
						return ErrUnmarshal
					}
				}
			case map[interface{}]interface{}:
				var t *task.Task

				if buf, err := yaml.Marshal(r); err == nil {
					if err := yaml.Unmarshal(buf, &t); err == nil {
						c.Tasks[k] = t
					} else {
						return ErrUnmarshal
					}
				} else {
					return ErrUnmarshal
				}
			}
		}

		return nil
	}

	return ErrUnmarshal
}

// ReadContent creates a new config struct from a string.
func ReadContent(content string) (*Config, error) {
	config := &Config{}
	config.Default()

	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, err
	}

	return config, nil
}

// ReadFile creates a new config struct from a yaml file.
func ReadFile(args ...string) (*Config, error) {
	var file string
	var path string
	var err error

	if len(args) > 0 && args[0] != "" {
		path = args[0]
	}

	var dat []byte

	if _, err := os.Stat(path); os.IsNotExist(err) {
		path, err = os.Getwd()

		files := []string{fmt.Sprintf("max_%s.yml", runtime.GOOS), "max.yml"}
		if len(file) > 0 {
			files = append([]string{file}, files...)
		}

		for _, name := range files {
			if len(dat) > 0 {
				break
			}

			file := filepath.Join(path, name)

			if _, err := os.Stat(file); err == nil {
				dat, err = ioutil.ReadFile(file)

				if err == nil {
					break
				}
			}
		}

	} else {
		dat, err = ioutil.ReadFile(path)
	}

	if err != nil {
		return nil, err
	}

	config := &Config{}
	config.Default()

	if err := yaml.Unmarshal(dat, &config); err != nil {
		return nil, err
	}

	return config, nil
}
