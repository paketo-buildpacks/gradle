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
	"github.com/paketo-buildpacks/libpak"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libbs"

	"github.com/paketo-buildpacks/gradle/gradle"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx         libcnb.BuildContext
		gradleBuild gradle.Build
		homeDir     string
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Buildpack.Metadata = map[string]interface{}{
			"configurations": []map[string]interface{}{
				{"name": "BP_GRADLE_BUILD_ARGUMENTS", "default": "test-argument"},
			},
		}

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		homeDir, err = ioutil.TempDir("", "home-dir")
		Expect(err).NotTo(HaveOccurred())

		gradleBuild = gradle.Build{
			ApplicationFactory: &FakeApplicationFactory{},
			HomeDirectoryResolver: FakeHomeDirectoryResolver{path: homeDir},
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())

		homeDir,err := gradleBuild.HomeDirectoryResolver.Location()
		Expect(err).ToNot(HaveOccurred())
		Expect(os.RemoveAll(homeDir)).To(Succeed())
	})

	it("does not contribute distribution if wrapper exists", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "gradlew"), []byte{}, 0644)).To(Succeed())
		ctx.StackID = "test-stack-id"

		result, err := gradleBuild.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		fi, err := os.Stat(filepath.Join(ctx.Application.Path, "gradlew"))
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode()).To(BeEquivalentTo(0755))

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("cache"))
		Expect(result.Layers[1].Name()).To(Equal("application"))
		Expect(result.Layers[1].(libbs.Application).Command).To(Equal(filepath.Join(ctx.Application.Path, "gradlew")))
	})

	it("contributes distribution", func() {
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


	context("gradle properties bindings exists", func() {
		var result libcnb.BuildResult

		it.Before(func() {
			var err error
			ctx.StackID = "test-stack-id"
			ctx.Platform.Path, err = ioutil.TempDir("", "gradle-test-platform")
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "gradlew"), []byte{}, 0644)).To(Succeed())
			ctx.Platform.Bindings = libcnb.Bindings{
				{
					Name:   "some-gradle",
					Type:   "gradle",
					Secret: map[string]string{"gradle.properties": "gradle-settings-content"},
					Path:   filepath.Join(ctx.Platform.Path, "bindings", "some-gradle"), // TODO: is this what get's hashed?
				},
			}
			gradleSettingsPath, ok := ctx.Platform.Bindings[0].SecretFilePath("gradle.properties")
			Expect(os.MkdirAll(filepath.Dir(gradleSettingsPath), 0777)).To(Succeed())
			Expect(ok).To(BeTrue())
			Expect(ioutil.WriteFile(
				gradleSettingsPath,
				[]byte("gradle-settings-content"),
				0644,
			)).To(Succeed())

			result, err = gradleBuild.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(2))
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Platform.Path)).To(Succeed())
		})

		it("provides gradle.properties under $GRADLE_USER_HOME", func() {
			gradlePropertiesPath := filepath.Join(homeDir, ".gradle", "gradle.properties")
			Expect(gradlePropertiesPath).To(BeARegularFile())

			data, err := ioutil.ReadFile(gradlePropertiesPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("gradle-settings-content"))
		})

		it("adds the hash of gradle.properties to the layer metadata", func() {
			md := result.Layers[1].(libbs.Application).LayerContributor.ExpectedMetadata
			mdMap, ok := md.(map[string]interface{})
			Expect(ok).To(BeTrue())
			// expected: sha256 of the string "gradle-settings-content"
			expected := "e6fdb059bdd9e59cec36afd5fb39c1e5b3c83694253b61c359701b4097520da4"
			Expect(mdMap["gradle-properties-sha256"]).To(Equal(expected))
		})
	})
}

type FakeApplicationFactory struct{}

func (f *FakeApplicationFactory) NewApplication(
	additionalMetdata map[string]interface{},
	argugments []string,
	_ libbs.ArtifactResolver,
	cache libbs.Cache,
	command string,
	_ *libcnb.BOM,
	_ string,
) (libbs.Application, error) {
	contributor := libpak.NewLayerContributor(
		"Compiled Application",
		additionalMetdata,
		libcnb.LayerTypes{Cache: true},
	)
	return libbs.Application{
		LayerContributor: contributor,
		Arguments:        argugments,
		Command:          command,
		Cache:            cache,
	}, nil
}

type FakeHomeDirectoryResolver struct{
	path string
}

func (f FakeHomeDirectoryResolver) Location() (string, error) {
	return f.path, nil
}