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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

const (
	PlanEntryGradle                = "gradle"
	PlanEntryJVMApplicationPackage = "jvm-application-package"
	PlaneEntryJDK                  = "jdk"
	PlanEntrySyft                  = "syft"
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {

	l := bard.NewLogger(ioutil.Discard)
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

		return libcnb.DetectResult{
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
						{Name: PlaneEntryJDK},
					},
				},
			},
		}, nil
	}

	return libcnb.DetectResult{Pass: false}, nil
}
