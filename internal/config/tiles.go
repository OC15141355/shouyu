package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Brand struct {
	Name          string `yaml:"name"`
	GreetingStyle string `yaml:"greeting_style"`
}

type Tile struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Href            string   `yaml:"href"`
	VisibleToGroups []string `yaml:"visible_to_groups"`
}

type Config struct {
	Brand Brand  `yaml:"brand"`
	Tiles []Tile `yaml:"tiles"`
}

// Load reads + validates the tiles YAML.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	for i, t := range c.Tiles {
		if t.ID == "" {
			return nil, fmt.Errorf("config: tile %d: id required", i)
		}
		if t.Name == "" {
			return nil, fmt.Errorf("config: tile %d (%s): name required", i, t.ID)
		}
		if t.Href == "" {
			return nil, fmt.Errorf("config: tile %d (%s): href required", i, t.ID)
		}
	}
	return &c, nil
}

// FilterByGroups returns tiles where any of userGroups overlaps with VisibleToGroups.
// Preserves input order.
func (c *Config) FilterByGroups(userGroups []string) []Tile {
	if c == nil {
		return nil
	}
	in := make(map[string]bool, len(userGroups))
	for _, g := range userGroups {
		in[g] = true
	}
	out := make([]Tile, 0, len(c.Tiles))
	for _, t := range c.Tiles {
		for _, g := range t.VisibleToGroups {
			if in[g] {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
