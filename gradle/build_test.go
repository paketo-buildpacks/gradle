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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/sbom"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libbs"

	"github.com/paketo-buildpacks/gradle/v7/gradle"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx             libcnb.BuildContext
		gradleBuild     gradle.Build
		homeDir         string
		gradlewFilepath string
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = os.MkdirTemp("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = os.MkdirTemp("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		homeDir, err = os.MkdirTemp("", "home-dir")
		Expect(err).NotTo(HaveOccurred())

		gradlewFilepath = filepath.Join(ctx.Application.Path, "gradlew")

		gradleBuild = gradle.Build{
			ApplicationFactory:    &FakeApplicationFactory{},
			HomeDirectoryResolver: FakeHomeDirectoryResolver{path: homeDir},
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(homeDir)).To(Succeed())
	})

	it("does not contribute distribution if wrapper exists", func() {
		Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())
		ctx.StackID = "test-stack-id"

		result, err := gradleBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("cache"))
		Expect(result.Layers[1].Name()).To(Equal("application"))
		Expect(result.Layers[1].(libbs.Application).Command).To(Equal(gradlewFilepath))
	})

	it("makes sure that gradlew is executable", func() {
		Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())
		ctx.StackID = "test-stack-id"

		_, err := gradleBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		fi, err := os.Stat(gradlewFilepath)
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode()).To(BeEquivalentTo(0755))
	})

	it("proceeds without error if gradlew could not have been made executable", func() {
		if _, err := os.Stat("/dev/null"); errors.Is(err, os.ErrNotExist) {
			t.Skip("No /dev/null thus not a unix system. Skipping chmod test.")
		}
		Expect(os.Symlink("/dev/null", gradlewFilepath)).To(Succeed())
		fi, err := os.Stat(gradlewFilepath)
		Expect(err).NotTo(HaveOccurred())
		originalMode := fi.Mode()
		Expect(originalMode).ToNot(BeEquivalentTo(0755))
		ctx.StackID = "test-stack-id"

		_, err = gradleBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		fi, err = os.Stat(gradlewFilepath)
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode()).To(BeEquivalentTo(originalMode))
	})

	it("contributes distribution for API <=0.6", func() {
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "gradle",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}
		ctx.StackID = "test-stack-id"
		ctx.Buildpack.API = "0.6"

		result, err := gradleBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("gradle"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
		Expect(result.Layers[2].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Layers.Path, "gradle", "bin", "gradle")))

		Expect(result.BOM.Entries).To(HaveLen(1))
		Expect(result.BOM.Entries[0].Name).To(Equal("gradle"))
		Expect(result.BOM.Entries[0].Build).To(BeTrue())
		Expect(result.BOM.Entries[0].Launch).To(BeFalse())
	})

	it("contributes distribution for API 0.7+", func() {
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "gradle",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
					"cpes":    []string{"cpe:2.3:a:apache:gradle:1.1.1:*:*:*:*:*:*:*"},
					"purl":    "pkg:generic/gradle@1.1.1",
				},
			},
		}
		ctx.StackID = "test-stack-id"

		result, err := gradleBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("gradle"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
		Expect(result.Layers[2].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Layers.Path, "gradle", "bin", "gradle")))

		Expect(result.BOM.Entries).To(HaveLen(1))
		Expect(result.BOM.Entries[0].Name).To(Equal("gradle"))
		Expect(result.BOM.Entries[0].Build).To(BeTrue())
		Expect(result.BOM.Entries[0].Launch).To(BeFalse())
	})

	context("BP_GRADLE_INIT_SCRIPT_PATH configuration is set", func() {
		it.Before(func() {
			ctx.Buildpack.Metadata = map[string]interface{}{
				"configurations": []map[string]interface{}{
					{"name": "BP_GRADLE_INIT_SCRIPT_PATH", "default": "/workspace/init.gradle"},
				},
			}
		})

		it("sets the settings path", func() {
			Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())

			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[1].(libbs.Application).Arguments).To(Equal([]string{
				"--init-script", "/workspace/init.gradle",
			}))
		})
	})

	context("BP_GRADLE_INIT_SCRIPT_PATH env var is set", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_GRADLE_INIT_SCRIPT_PATH", "/workspace/init.gradle")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv(("BP_GRADLE_INIT_SCRIPT_PATH"))).To(Succeed())
		})

		it("sets the settings path", func() {
			Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())

			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[1].(libbs.Application).Arguments).To(Equal([]string{
				"--init-script", "/workspace/init.gradle",
			}))
		})
	})

	context("BP_GRADLE_BUILD_ARGUMENTS env var is set", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_GRADLE_BUILD_ARGUMENTS", "--no-daemon -Dorg.gradle.welcome=never assemble")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv(("BP_GRADLE_BUILD_ARGUMENTS"))).To(Succeed())
		})
		it("sets some build arguments", func() {
			Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())

			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[1].(libbs.Application).Arguments).To(Equal([]string{
				"--no-daemon", "-Dorg.gradle.welcome=never", "assemble",
			}))
		})
	})

	context("BP_GRADLE_BUILD_ARGUMENTS and BP_GRADLE_ADDITIONAL_BUILD_ARGUMENTS  env var is set", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_GRADLE_BUILD_ARGUMENTS", "--no-daemon assemble")).To(Succeed())
			Expect(os.Setenv("BP_GRADLE_ADDITIONAL_BUILD_ARGUMENTS", "--no-build-cache")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv(("BP_GRADLE_BUILD_ARGUMENTS"))).To(Succeed())
			Expect(os.Unsetenv(("BP_GRADLE_ADDITIONAL_BUILD_ARGUMENTS"))).To(Succeed())
		})
		it("sets some build and additional build arguments", func() {
			Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())

			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[1].(libbs.Application).Arguments).To(Equal([]string{
				"--no-daemon", "assemble", "--no-build-cache",
			}))
		})
	})

	context("gradle properties bindings exists", func() {
		var bindingPath string

		it.Before(func() {
			var err error
			ctx.StackID = "test-stack-id"
			ctx.Platform.Path, err = os.MkdirTemp("", "gradle-test-platform")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())
			bindingPath = filepath.Join(ctx.Platform.Path, "bindings", "some-gradle")
			ctx.Platform.Bindings = libcnb.Bindings{
				{
					Name:   "some-gradle",
					Type:   "gradle",
					Secret: map[string]string{"gradle.properties": "gradle-properties-content"},
					Path:   bindingPath,
				},
			}
			gradlePropertiesPath, ok := ctx.Platform.Bindings[0].SecretFilePath("gradle.properties")
			Expect(os.MkdirAll(filepath.Dir(gradlePropertiesPath), 0777)).To(Succeed())
			Expect(ok).To(BeTrue())
			Expect(os.WriteFile(
				gradlePropertiesPath,
				[]byte("gradle-properties-content"),
				0644,
			)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Platform.Path)).To(Succeed())
		})

		it("provides gradle.properties under $GRADLE_USER_HOME", func() {
			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))
			Expect(result.Layers[1].Name()).To(Equal("gradle-properties"))
			Expect(result.Layers[1])
		})

		it("adds the hash of gradle.properties to the layer metadata", func() {
			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			md := result.Layers[2].(libbs.Application).LayerContributor.ExpectedMetadata
			mdMap, ok := md.(map[string]interface{})
			Expect(ok).To(BeTrue())
			// expected: sha256 of the string "gradle-properties-content"
			expected := "6621087fb513e8db5544d34ccad59720793a1a5a9eb20a2df554422b8b5e50e5"
			Expect(mdMap["gradle-properties-sha256"]).To(Equal(expected))
		})
	})

	context("gradle wrapper properties binding exists", func() {
		var bindingPath string

		it.Before(func() {
			var err error
			ctx.StackID = "test-stack-id"
			ctx.Platform.Path, err = os.MkdirTemp("", "gradle-test-platform")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(gradlewFilepath, []byte{}, 0644)).To(Succeed())
			bindingPath = filepath.Join(ctx.Platform.Path, "bindings", "some-gradle")
			ctx.Platform.Bindings = libcnb.Bindings{
				{
					Name:   "some-gradle",
					Type:   "gradle-wrapper",
					Secret: map[string]string{"gradle-wrapper.properties": "gradle-wrapper-properties-content"},
					Path:   bindingPath,
				},
			}
			gradlePropertiesPath, ok := ctx.Platform.Bindings[0].SecretFilePath("gradle-wrapper.properties")
			Expect(os.MkdirAll(filepath.Dir(gradlePropertiesPath), 0777)).To(Succeed())
			Expect(ok).To(BeTrue())
			Expect(os.WriteFile(
				gradlePropertiesPath,
				[]byte("gradle-wrapper-properties-content"),
				0644,
			)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Platform.Path)).To(Succeed())
		})

		it("contributes bound gradle-wrapper.properties", func() {
			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))
			Expect(result.Layers[1].Name()).To(Equal("gradle-wrapper-properties"))
			Expect(result.Layers[1])
		})

		it("adds the hash of gradle-wrapper.properties to the layer metadata", func() {
			result, err := gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			md := result.Layers[2].(libbs.Application).LayerContributor.ExpectedMetadata
			mdMap, ok := md.(map[string]interface{})
			Expect(ok).To(BeTrue())
			// expected: sha256 of the string "gradle-wrapper-properties-content"
			expected := "8d98502ceb9504c887b12cfba9427c5338d133a2f10613cb0137695ca09c7ddc"
			Expect(mdMap["gradle-wrapper-properties-sha256"]).To(Equal(expected))
		})
	})

}

type FakeApplicationFactory struct{}

func (f *FakeApplicationFactory) NewApplication(
	additionalMetdata map[string]interface{},
	args []string,
	_ libbs.ArtifactResolver,
	_ libbs.Cache,
	command string,
	_ *libcnb.BOM,
	_ string,
	_ sbom.SBOMScanner,
) (libbs.Application, error) {
	contributor := libpak.NewLayerContributor(
		"Compiled Application",
		additionalMetdata,
		libcnb.LayerTypes{Cache: true},
	)
	return libbs.Application{
		LayerContributor: contributor,
		Command:          command,
		Arguments:        args,
	}, nil
}

type FakeHomeDirectoryResolver struct {
	path string
}

func (f FakeHomeDirectoryResolver) Location() (string, error) {
	return f.path, nil
}
