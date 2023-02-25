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

package gradle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/gradle/v6/gradle"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect gradle.Detect
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = os.MkdirTemp("", "gradle")
		Expect(err).NotTo(HaveOccurred())
		os.Unsetenv("BP_GRADLE_BUILD_FILE")
		os.Unsetenv("BP_JAVA_INSTALL_NODE")
		os.Unsetenv("BP_NODE_PROJECT_PATH")
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
	})

	it("fails without build.gradle or build.gradle.kts", func() {
		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{}))
	})

	it("fails without configured build file", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		os.Setenv("BP_GRADLE_BUILD_FILE", filepath.Join(ctx.Application.Path, "no-such-build.gradle"))
		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{}))
	})

	it("passes if no package-manager file is found", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
					},
				},
			},
		}))
	})

	it("passes with build.gradle", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
					},
				},
			},
		}))
	})

	it("passes with build.gradle.kts", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle.kts"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
					},
				},
			},
		}))
	})

	it("passes with settings.gradle", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "settings.gradle"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
					},
				},
			},
		}))
	})

	it("passes with settings.gradle.kts", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "settings.gradle.kts"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
					},
				},
			},
		}))
	})

	it("passes with package.json", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "package.json"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("passes with yarn.lock", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "yarn.lock"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("passes without duplication with both yarn.lock & package.json", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "yarn.lock"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "package.json"), []byte{}, 0644))
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("passes with custom path set via BP_NODE_PROJECT_PATH", func() {
		os.Setenv("BP_NODE_PROJECT_PATH",  "frontend")
		os.Setenv("BP_JAVA_INSTALL_NODE",  "true")
		os.Mkdir(filepath.Join(ctx.Application.Path, "frontend"), 0755)
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "frontend/yarn.lock"), []byte{}, 0644))


		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
						{Name: "yarn", Metadata: map[string]interface{}{"build": true}},
						{Name: "node", Metadata: map[string]interface{}{"build": true}},
					},
				},
			},
		}))
	})

	it("does not detect false positive without env-var", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "yarn.lock"), []byte{}, 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "package.json"), []byte{}, 0644))

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "gradle"},
						{Name: "jvm-application-package"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "syft"},
						{Name: "gradle"},
						{Name: "jdk"},
					},
				},
			},
		}))
	})
}
