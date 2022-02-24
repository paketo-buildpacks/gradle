package gradle_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/gradle/v6/gradle"
)

func testGradleProperties(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx                   libcnb.BuildContext
		gradleProps           gradle.PropertiesFile
		gradleLayer           libcnb.Layer
		gradleHome            string
		gradleTargetPropsPath string
		bindingPath           string
		homeDir               string
	)

	it.Before(func() {
		var err error

		ctx.Platform.Path, err = ioutil.TempDir("", "gradle-test-platform")
		Expect(err).NotTo(HaveOccurred())

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		homeDir, err = ioutil.TempDir("", "home-dir")
		Expect(err).NotTo(HaveOccurred())

		gradleHome = filepath.Join(homeDir, ".gradle")
		gradleTargetPropsPath = filepath.Join(gradleHome, "gradle.properties")
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

	context("a binding is present", func() {
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
			Expect(ioutil.WriteFile(
				gradleSrcPropsPath,
				[]byte("gradle-properties-content"),
				0644,
			)).To(Succeed())

			// normally done by cache layer
			Expect(os.MkdirAll(gradleHome, 0755)).ToNot(HaveOccurred())

			gradleLayer, err = ctx.Layers.Layer("gradle-properties")
			Expect(err).NotTo(HaveOccurred())

			gradleProps = gradle.PropertiesFile{
				Binding:    ctx.Platform.Bindings[0],
				GradleHome: gradleHome,
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

			data, err := ioutil.ReadFile(gradleTargetPropsPath)
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

			data, err := ioutil.ReadFile(gradleTargetPropsPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("gradle-properties-content"))
		})

	})
}
