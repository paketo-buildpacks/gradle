/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gradle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/paketo-buildpacks/libpak/bindings"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libbs"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger                bard.Logger
	ApplicationFactory    ApplicationFactory
	HomeDirectoryResolver HomeDirectoryResolver
}

type ApplicationFactory interface {
	NewApplication(additionalMetadata map[string]interface{}, arguments []string, artifactResolver libbs.ArtifactResolver,
		cache libbs.Cache, command string, bom *libcnb.BOM, applicationPath string) (libbs.Application, error)
}

type HomeDirectoryResolver interface {
	Location() (string, error)
}

type OSHomeDirectoryResolver struct{}

func (p OSHomeDirectoryResolver) Location() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory\n%w", err)
	}

	return u.HomeDir, nil
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	command := filepath.Join(context.Application.Path, "gradlew")
	if _, err := os.Stat(command); os.IsNotExist(err) {
		dep, err := dr.Resolve("gradle", "")
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		d, be := NewDistribution(dep, dc)
		d.Logger = b.Logger
		result.Layers = append(result.Layers, d)
		result.BOM.Entries = append(result.BOM.Entries, be)

		command = filepath.Join(context.Layers.Path, d.Name(), "bin", "gradle")
	} else if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to stat %s\n%w", command, err)
	} else {
		if err := os.Chmod(command, 0755); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to chmod %s\n%w", command, err)
		}
	}

	homeDir, err := b.HomeDirectoryResolver.Location()
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve home directory\n%w", err)
	}
	gradleHome := filepath.Join(homeDir, ".gradle")

	c := libbs.Cache{Path: gradleHome}
	c.Logger = b.Logger
	result.Layers = append(result.Layers, c)

	args, err := libbs.ResolveArguments("BP_GRADLE_BUILD_ARGUMENTS", cr)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve build arguments\n%w", err)
	}

	md := map[string]interface{}{}
	if binding, ok, err := bindings.ResolveOne(context.Platform.Bindings, bindings.OfType("gradle")); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve binding\n%w", err)
	} else if ok {
		err = handleGradleSettings(binding, gradleHome, md)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to process maven settings from binding\n%w", err)
		}
	}

	art := libbs.ArtifactResolver{
		ArtifactConfigurationKey: "BP_GRADLE_BUILT_ARTIFACT",
		ConfigurationResolver:    cr,
		ModuleConfigurationKey:   "BP_GRADLE_BUILT_MODULE",
		InterestingFileDetector:  libbs.JARInterestingFileDetector{},
	}

	a, err := b.ApplicationFactory.NewApplication(
		md,
		args,
		art,
		c,
		command,
		result.BOM,
		context.Application.Path,
	)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create application layer\n%w", err)
	}
	a.Logger = b.Logger
	result.Layers = append(result.Layers, a)

	return result, nil
}

func handleGradleSettings(binding libcnb.Binding, gradleHome string, md map[string]interface{}) error {
	path, ok := binding.SecretFilePath("gradle.properties")
	if !ok {
		return nil
	}

	err := os.MkdirAll(gradleHome, 0755)
	if err != nil {
		return fmt.Errorf("unable to make gradle home\n%w", err)
	}

	gradlePropertiesPath := filepath.Join(gradleHome, "gradle.properties")
	if err := os.Symlink(path, gradlePropertiesPath); os.IsExist(err) {
		err = os.Remove(gradlePropertiesPath)
		if err != nil {
			return fmt.Errorf("unable to remove old symlink for gradle.properties\n%w", err)
		}

		err = os.Symlink(path, gradlePropertiesPath)
		if err != nil {
			return fmt.Errorf("unable to create symlink for gradle.properties on retry\n%w", err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to symlink bound gradle.properties\n%w", err)
	}

	hasher := sha256.New()
	propertiesFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open gradle.properties\n%w", err)
	}
	if _, err := io.Copy(hasher, propertiesFile); err != nil {
		return fmt.Errorf("error hashing gradle.properties\n%w", err)
	}

	md["gradle-properties-sha256"] = hex.EncodeToString(hasher.Sum(nil))
	return nil
}
