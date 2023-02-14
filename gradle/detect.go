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
	"fmt"
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"os"
	"path/filepath"
	"strings"
)

const (
	PlanEntryGradle                = "gradle"
	PlanEntryJVMApplicationPackage = "jvm-application-package"
	PlanEntryJDK                   = "jdk"
	PlanEntrySyft                  = "syft"
	PlanEntryYarn				   = "yarn"
	PlanEntryNode				   = "node"
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {

	result := libcnb.DetectResult{}
	l := bard.NewLogger(os.Stdout)
	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &l)
	if err != nil {
		return libcnb.DetectResult{}, err
	}

	buildFile, _ := cr.Resolve("BP_GRADLE_BUILD_FILE")

	if buildFile != "" {

		file := filepath.Join(context.Application.Path, buildFile)
		_, err = os.Stat(file)
		if os.IsNotExist(err) {
			return libcnb.DetectResult{Pass: false}, nil
		} else if err != nil {
			return libcnb.DetectResult{}, fmt.Errorf("unable to determine if %s exists\n%w", file, err)
		}

	}

	var files []string
	if buildFile != "" {
		files = []string{
			filepath.Join(context.Application.Path, buildFile),
		}
	} else {
		files = []string{
			filepath.Join(context.Application.Path, "build.gradle"),
			filepath.Join(context.Application.Path, "build.gradle.kts"),
			filepath.Join(context.Application.Path, "settings.gradle"),
			filepath.Join(context.Application.Path, "settings.gradle.kts"),
		}
	}

	for _, file := range files {
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return libcnb.DetectResult{}, fmt.Errorf("unable to determine if %s exists\n%w", file, err)
		}

		result = libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: PlanEntryGradle},
						{Name: PlanEntryJVMApplicationPackage},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: PlanEntrySyft},
						{Name: PlanEntryGradle},
						{Name: PlanEntryJDK},
					},
				},
			},
		}
	}
	// Gradle detection has passed
	if len(result.Plans) > 0 {
		jsFile, _ := cr.Resolve("BP_JS_PACKAGE_MANAGER_FILE")
		if jsFile != "" {
					file := filepath.Join(context.Application.Path, jsFile)
					_, err := os.Stat(file)
					if os.IsNotExist(err) {
						l.Infof("unable to find package manager file %s, specified in BP_JS_PACKAGE_MANAGER_FILE", file)
						return result, nil
					} else if err != nil {
						return libcnb.DetectResult{}, fmt.Errorf("unable to determine if file %s exists \n%w", file, err)
					}
					if strings.Contains(file, "yarn.lock") {
						result.Plans[0].Requires = append(result.Plans[0].Requires, libcnb.BuildPlanRequire{Name: PlanEntryYarn, Metadata: map[string]interface{}{"build": true}})
						result.Plans[0].Requires = append(result.Plans[0].Requires, libcnb.BuildPlanRequire{Name: PlanEntryNode, Metadata: map[string]interface{}{"build": true}})
					} else if strings.Contains(file, "package.json") {
						result.Plans[0].Requires = append(result.Plans[0].Requires, libcnb.BuildPlanRequire{Name: PlanEntryNode, Metadata: map[string]interface{}{"build": true}})
					}
		}
		return result, nil
	}

	return libcnb.DetectResult{Pass: false}, nil
}
