package gradle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
)

type PropertiesFile struct {
	Binding    libcnb.Binding
	GradleHome string
}

func (p PropertiesFile) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	path, ok := p.Binding.SecretFilePath("gradle.properties")
	if !ok {
		return layer, nil
	}

	gradlePropertiesPath := filepath.Join(p.GradleHome, "gradle.properties")
	if err := os.Symlink(path, gradlePropertiesPath); os.IsExist(err) {
		err = os.Remove(gradlePropertiesPath)
		if err != nil {
			return layer, fmt.Errorf("unable to remove old symlink for gradle.properties\n%w", err)
		}

		err = os.Symlink(path, gradlePropertiesPath)
		if err != nil {
			return layer, fmt.Errorf("unable to create symlink for gradle.properties on retry\n%w", err)
		}
	} else if err != nil {
		return layer, fmt.Errorf("unable to symlink bound gradle.properties\n%w", err)
	}

	return layer, nil
}

func (p PropertiesFile) Name() string {
	return "gradle-properties"
}
