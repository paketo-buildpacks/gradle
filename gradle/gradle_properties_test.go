package gradle_test

import (
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/gradle/v7/gradle"
)

func testGradleProperties(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx                          libcnb.BuildContext
		gradleProps                  gradle.PropertiesFile
		gradleLayer                  libcnb.Layer
		gradleHome                   string
		gradleWrapperHome            string
		gradleWrapperTargetPropsPath string
		gradleTargetPropsPath        string
		bindingPath                  string
		homeDir                      string
	)

	it.Before(func() {
		var err error

		ctx.Platform.Path, err = os.MkdirTemp("", "gradle-test-platform")
		Expect(err).NotTo(HaveOccurred())

		ctx.Application.Path, err = os.MkdirTemp("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = os.MkdirTemp("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		homeDir, err = os.MkdirTemp("", "home-dir")
		Expect(err).NotTo(HaveOccurred())

		gradleHome = filepath.Join(homeDir, ".gradle")
		gradleTargetPropsPath = filepath.Join(gradleHome, "gradle.properties")

		gradleWrapperHome = filepath.Join(ctx.Application.Path, "gradle", "wrapper")
		gradleWrapperTargetPropsPath = filepath.Join(gradleWrapperHome, "gradle-wrapper.properties")
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Platform.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(homeDir)).To(Succeed())
	})

	context("no binding is present", func() {
		it("does nothing ", func() {
			layer, err := gradleProps.Contribute(gradleLayer)
			Expect(err).NotTo(HaveOccurred())
			Expect(layer).To(Equal(gradleLayer))

			Expect(gradleHome).ToNot(BeADirectory())
			Expect(gradleTargetPropsPath).ToNot(BeAnExistingFile())
		})
	})

	context("a gradle properties binding is present", func() {
		it.Before(func() {
			var err error

			bindingPath = filepath.Join(ctx.Platform.Path, "bindings", "some-gradle")
			ctx.Platform.Bindings = libcnb.Bindings{
				{
					Name:   "some-gradle",
					Type:   "gradle",
					Secret: map[string]string{"gradle.properties": "gradle-properties-content"},
					Path:   bindingPath,
				},
			}
			gradleSrcPropsPath, ok := ctx.Platform.Bindings[0].SecretFilePath("gradle.properties")
			Expect(os.MkdirAll(filepath.Dir(gradleSrcPropsPath), 0777)).To(Succeed())
			Expect(ok).To(BeTrue())
			Expect(os.WriteFile(
				gradleSrcPropsPath,
				[]byte("gradle-properties-content"),
				0644,
			)).To(Succeed())

			// normally done by cache layer
			Expect(os.MkdirAll(gradleHome, 0755)).ToNot(HaveOccurred())

			gradleLayer, err = ctx.Layers.Layer("gradle-properties")
			Expect(err).NotTo(HaveOccurred())

			gradleProps = gradle.PropertiesFile{
				Binding:                  ctx.Platform.Bindings[0],
				GradlePropertiesHome:     gradleHome,
				GradlePropertiesFileName: "gradle.properties",
				GradlePropertiesName:     "gradle-properties",
			}
		})

		it("creates a symlink for gradle.properties under $GRADLE_USER_HOME", func() {
			layer, err := gradleProps.Contribute(gradleLayer)
			Expect(err).NotTo(HaveOccurred())
			Expect(layer).To(Equal(gradleLayer))

			info, err := os.Lstat(gradleTargetPropsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()&os.ModeSymlink != 0).To(BeTrue()) // is symlink bit set

			target, err := os.Readlink(gradleTargetPropsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(Equal(filepath.Join(bindingPath, "gradle.properties")))

			data, err := os.ReadFile(gradleTargetPropsPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("gradle-properties-content"))
		})

		it("recreates symlink for gradle.properties under $GRADLE_USER_HOME", func() {
			Expect(os.MkdirAll(filepath.Dir(gradleTargetPropsPath), 0755)).ToNot(HaveOccurred())
			err := os.Symlink("/dev/null", gradleTargetPropsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(gradleTargetPropsPath).To(BeAnExistingFile())

			layer, err := gradleProps.Contribute(gradleLayer)
			Expect(err).NotTo(HaveOccurred())
			Expect(layer).To(Equal(gradleLayer))

			info, err := os.Lstat(gradleTargetPropsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()&os.ModeSymlink != 0).To(BeTrue()) // is symlink bit set

			target, err := os.Readlink(gradleTargetPropsPath)
			Expect(err).NotTo(HaveOccurred())
			// symlink should point to our binding, not /dev/null
			Expect(target).To(Equal(filepath.Join(bindingPath, "gradle.properties")))

			data, err := os.ReadFile(gradleTargetPropsPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("gradle-properties-content"))
		})

	})

	context("a gradle wrapper properties binding is present and contributes its content", func() {
		it.Before(func() {
			var err error

			bindingPath = filepath.Join(ctx.Platform.Path, "bindings", "some-gradle")
			ctx.Platform.Bindings = libcnb.Bindings{
				{
					Name: "some-gradle",
					Type: "gradle-wrapper",
					Secret: map[string]string{"gradle-wrapper.properties": `distributionUrl=https://g.o/gradle-7.5-bin.zip
						 networkTimeout=43`},
					Path: bindingPath,
				},
			}
			Expect(os.MkdirAll(gradleWrapperHome, 0755)).ToNot(HaveOccurred())
			gradleSrcPropsPath, ok := ctx.Platform.Bindings[0].SecretFilePath("gradle-wrapper.properties")
			Expect(os.MkdirAll(filepath.Dir(gradleSrcPropsPath), 0777)).To(Succeed())
			Expect(ok).To(BeTrue())

			Expect(os.WriteFile(
				gradleSrcPropsPath,
				[]byte(`distributionUrl=https://g.o/gradle-7.5-bin.zip
						 networkTimeout=43`),
				0644,
			)).To(Succeed())

			originalGradleWrapperProperties, err := os.Open(filepath.Join("testdata", "gradle-wrapper.properties"))
			Expect(err).NotTo(HaveOccurred())

			err = sherpa.CopyFile(originalGradleWrapperProperties, gradleWrapperTargetPropsPath)
			Expect(err).NotTo(HaveOccurred())

			gradleLayer, err = ctx.Layers.Layer("gradle-wrapper-properties")
			Expect(err).NotTo(HaveOccurred())

			gradleProps = gradle.PropertiesFile{
				Binding:                  ctx.Platform.Bindings[0],
				GradlePropertiesHome:     gradleWrapperHome,
				GradlePropertiesFileName: "gradle-wrapper.properties",
				GradlePropertiesName:     "gradle-wrapper-properties",
			}
		})

		it("merges gradle wrapper properties files", func() {
			layer, err := gradleProps.Contribute(gradleLayer)
			Expect(err).NotTo(HaveOccurred())
			Expect(layer).To(Equal(gradleLayer))

			patchedProperties := properties.MustLoadFile(filepath.Join("testdata", "gradle-wrapper.properties"), properties.UTF8)
			previousValue, ok, err := patchedProperties.Set("networkTimeout", "43")
			Expect(err).NotTo(HaveOccurred())
			Expect(previousValue).To(Equal("10000"))
			Expect(ok).To(Equal(true))
			previousValue, ok, err = patchedProperties.Set("distributionUrl", "https://g.o/gradle-7.5-bin.zip")
			Expect(err).NotTo(HaveOccurred())
			Expect(previousValue).To(Equal("https://services.gradle.org/distributions/gradle-7.6-bin.zip"))
			Expect(ok).To(Equal(true))

			finalProperties := properties.MustLoadFile(gradleWrapperTargetPropsPath, properties.UTF8)

			Expect(finalProperties).To(Equal(patchedProperties))
		})

	})

}
