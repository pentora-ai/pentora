package fingerprint

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ValidationDataset represents the structure of validation_dataset.yaml.
type ValidationDataset struct {
	TruePositives []ValidationTestCase `yaml:"true_positives"`
	TrueNegatives []ValidationTestCase `yaml:"true_negatives"`
	EdgeCases     []ValidationTestCase `yaml:"edge_cases"`
}

// ValidationTestCase represents a single labeled validation scenario.
type ValidationTestCase struct {
	Protocol        string `yaml:"protocol"`
	Port            int    `yaml:"port"`
	Banner          string `yaml:"banner"`
	ExpectedProduct string `yaml:"expected_product,omitempty"`
	ExpectedVendor  string `yaml:"expected_vendor,omitempty"`
	ExpectedVersion string `yaml:"expected_version,omitempty"`
	ExpectedMatch   *bool  `yaml:"expected_match,omitempty"`
	Description     string `yaml:"description"`
}

// LoadValidationDataset loads and parses a validation dataset YAML file.
func LoadValidationDataset(path string) (*ValidationDataset, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read validation dataset: %w", err)
	}
	var ds ValidationDataset
	if err := yaml.Unmarshal(b, &ds); err != nil {
		return nil, fmt.Errorf("failed to parse validation dataset: %w", err)
	}
	return &ds, nil
}
