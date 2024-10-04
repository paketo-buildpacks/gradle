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

	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sbom"

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
		cache libbs.Cache, command string, bom *libcnb.BOM, applicationPath string, bomScanner sbom.SBOMScanner) (libbs.Application, error)
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
			b.Logger.Bodyf("WARNING: unable to chmod %s:\n%s", command, err)
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

	additionalArgs, err := libbs.ResolveArguments("BP_GRADLE_ADDITIONAL_BUILD_ARGUMENTS", cr)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve additional build arguments\n%w", err)
	} else {
		args = append(args, additionalArgs...)
	}

	initScriptPath, _ := cr.Resolve("BP_GRADLE_INIT_SCRIPT_PATH")
	if initScriptPath != "" {
		args = append([]string{"--init-script", initScriptPath}, args...)
	}

	md := map[string]interface{}{}
	if binding, ok, err := bindings.ResolveOne(context.Platform.Bindings, bindings.OfType("gradle")); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve binding\n%w", err)
	} else if ok {
		b.Logger.Debug("binding of type gradle successfully detected, configuring layer")
		gradlePropertiesPath, ok := binding.SecretFilePath("gradle.properties")
		if ok {
			gradlePropertiesFile, err := os.Open(gradlePropertiesPath)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to open gradle.properties\n%w", err)
			}

			hasher := sha256.New()
			if _, err := io.Copy(hasher, gradlePropertiesFile); err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to hash gradle.properties\n%w", err)
			}
			md["gradle-properties-sha256"] = hex.EncodeToString(hasher.Sum(nil))

			result.Layers = append(result.Layers, PropertiesFile{
				binding,
				gradleHome,
				"gradle.properties",
				"gradle-properties",
				b.Logger,
			})
		}
	}

	gradleWrapperHome := filepath.Join("gradle", "wrapper")
	if binding, ok, err := bindings.ResolveOne(context.Platform.Bindings, bindings.OfType("gradle-wrapper")); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve binding\n%w", err)
	} else if ok {
		b.Logger.Debug("binding of type gradle-wrapper successfully detected, configuring layer")
		gradleWrapperPropertiesPath, ok := binding.SecretFilePath("gradle-wrapper.properties")
		if ok {
			gradleWrapperPropertiesFile, err := os.Open(gradleWrapperPropertiesPath)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to open gradle-wrapper.properties\n%w", err)
			}

			hasher := sha256.New()
			if _, err := io.Copy(hasher, gradleWrapperPropertiesFile); err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to hash gradle-wrapper.properties\n%w", err)
			}
			md["gradle-wrapper-properties-sha256"] = hex.EncodeToString(hasher.Sum(nil))

			result.Layers = append(result.Layers, PropertiesFile{
				binding,
				gradleWrapperHome,
				"gradle-wrapper.properties",
				"gradle-wrapper-properties",
				b.Logger,
			})
		}
	}

	art := libbs.ArtifactResolver{
		ArtifactConfigurationKey: "BP_GRADLE_BUILT_ARTIFACT",
		ConfigurationResolver:    cr,
		ModuleConfigurationKey:   "BP_GRADLE_BUILT_MODULE",
		InterestingFileDetector:  libbs.JARInterestingFileDetector{},
		AdditionalHelpMessage:    "If this is unexpected, please try setting `rootProject.name` in `settings.gradle` or add a project.toml file and exclude the `build/` directory. For details see https://buildpacks.io/docs/app-developer-guide/using-project-descriptor/.",
	}

	bomScanner := sbom.NewSyftCLISBOMScanner(context.Layers, effect.CommandExecutor{}, b.Logger)
	a, err := b.ApplicationFactory.NewApplication(
		md,
		args,
		art,
		c,
		command,
		result.BOM,
		context.Application.Path,
		bomScanner,
	)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create application layer\n%w", err)
	}
	a.Logger = b.Logger
	result.Layers = append(result.Layers, a)

	return result, nil
}
