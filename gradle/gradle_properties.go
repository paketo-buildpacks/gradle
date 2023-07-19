package gradle

import (
	"fmt"
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libpak/bard"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
)

type PropertiesFile struct {
	Binding                  libcnb.Binding
	GradlePropertiesHome     string
	GradlePropertiesFileName string
	GradlePropertiesName     string
	Logger                   bard.Logger
}

func (p PropertiesFile) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	path, ok := p.Binding.SecretFilePath(p.GradlePropertiesFileName)
	if !ok {
		return libcnb.Layer{}, nil
	}

	originalPropertiesFilePath := filepath.Join(p.GradlePropertiesHome, p.GradlePropertiesFileName)
	if p.GradlePropertiesName == "gradle-properties" {
		p.Logger.Debug("symlinking gradle-properties bound file")
		gradlePropertiesPath := originalPropertiesFilePath
		if err := os.Symlink(path, gradlePropertiesPath); os.IsExist(err) {
			err = os.Remove(gradlePropertiesPath)
			if err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to remove old symlink for %s\n%w", p.GradlePropertiesFileName, err)
			}

			err = os.Symlink(path, gradlePropertiesPath)
			if err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to create symlink for %s on retry\n%w", p.GradlePropertiesFileName, err)
			}
		} else if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to symlink bound %s\n%w", p.GradlePropertiesFileName, err)
		}
	} else if p.GradlePropertiesName == "gradle-wrapper-properties" {
		file, err := os.ReadFile(path)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to read bound gradle-wrapper.properties file at %s\n%w", path, err)
		}
		p.Logger.Debugf("applying these bound gradle-wrapper-properties to default one: \n%s\n", string(file))
		mergedProperties := properties.MustLoadFiles([]string{originalPropertiesFilePath, path}, properties.UTF8, true)
		propertiesFile, err := os.Create(originalPropertiesFilePath)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to create/update original gradle-wrapper.properties file at %s\n%w", originalPropertiesFilePath, err)
		}
		_, err = mergedProperties.Write(propertiesFile, properties.UTF8)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to merge gradle-wrapper.properties files.\n%w", err)
		}
	}

	return layer, nil
}

func (p PropertiesFile) Name() string {
	return p.GradlePropertiesName
}
