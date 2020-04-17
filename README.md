# `gcr.io/paketo-buildpacks/gradle`
The Paketo <NAME> Buildpack is a Cloud Native Buildpack that build Gradle-based applications from source.

## Behavior
This buildpack will participate ant of the following conditions are met

* `<APPLICATION_ROOT>/build.gradle` exists
* `<APPLICATION_ROOT>/build.gradle.kts` exists

The buildpack will do the following:

* Requests that a JDK be installed
* Links the `~/.gradle` to a layer for caching
* If `<APPLICATION_ROOT>/gradlew` exists
  * Runs `<APPLICATION_ROOT>/gradlew --no-daemon -x test build` to build the application
* If `<APPLICATION_ROOT>/gradlew` does not exist
  * Contributes Gradle to a layer with all commands on `$PATH`
  * Runs `<GRADLE_ROOT>/gradle -x test build` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* Expands `<APPLICATION_ROOT>/build/libs/*.[jw]ar` to `<APPLICATION_ROOT>`

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_GRADLE_BUILD_ARGUMENTS` | Configure the arguments to pass to build system.  Defaults to `--no-daemon -x test build`.
| `$BP_GRADLE_BUILT_MODULE` | Configure the module to find application artifact in.  Defaults to the root module (empty).
| `$BP_GRADLE_BUILT_ARTIFACT` | Configure the built application artifact explicitly.  Supersedes `$BP_BUILT_MODULE`  Defaults to `build/libs/*.[jw]ar`.

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
