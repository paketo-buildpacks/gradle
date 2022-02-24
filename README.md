# `gcr.io/paketo-buildpacks/gradle`

The Paketo Gradle Buildpack is a Cloud Native Buildpack that build Gradle-based applications from source.

## Behavior

This buildpack will participate if any of the following conditions are met

* `<APPLICATION_ROOT>/build.gradle` exists
* `<APPLICATION_ROOT>/build.gradle.kts` exists

The buildpack will do the following:

* Requests that a JDK be installed
* Links the `~/.gradle` to a layer for caching
* If `<APPLICATION_ROOT>/gradlew` exists
  * Runs `<APPLICATION_ROOT>/gradlew --no-daemon assemble` to build the application
* If `<APPLICATION_ROOT>/gradlew` does not exist
  * Contributes Gradle to a layer with all commands on `$PATH`
  * Runs `<GRADLE_ROOT>/bin/gradle --no-daemon assemble` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* If `$BP_GRADLE_BUILT_ARTIFACT` matched a single file
  * Restores `$BP_GRADLE_BUILT_ARTIFACT` from the layer, expands the single file to `<APPLICATION_ROOT>`
* If `$BP_GRADLE_BUILT_ARTIFACT` matched a directory or multiple files
  * Restores the files matched by `$BP_GRADLE_BUILT_ARTIFACT` to `<APPLICATION_ROOT>`

## Configuration

| Environment Variable         | Description                                                                                                                                                                                                                          |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `$BP_GRADLE_BUILD_ARGUMENTS` | Configure the arguments to pass to build system. Defaults to `--no-daemon assemble`.                                                                                                                                                 |
| `$BP_GRADLE_BUILD_FILE`      | Configure the location of the build configuration file. If it doesn't exist this build pack will not be applied. Defaults to `build.gradle`.                                                                                         |
| `$BP_GRADLE_BUILT_MODULE`    | Configure the module to find application artifact in. Defaults to the root module (empty).                                                                                                                                           |
| `$BP_GRADLE_BUILT_ARTIFACT`  | Configure the built application artifact explicitly. Supersedes `$BP_GRADLE_BUILT_ARTIFACT`. Defaults to `build/libs/*.[jw]ar`. Can match a single file, multiple files or a directory. Can be one or more space separated patterns. |

## Bindings

The buildpack optionally accepts the following bindings:

### Type: `gradle`

| Secret              | Description                                                                                                                                                                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `gradle.properties` | If present, the contents of the file are copied to `$GRADLE_USER_HOME/gradle.properties` which is [picked up by gradle and merged](https://docs.gradle.org/current/userguide/build_environment.html#sec:gradle_configuration_properties) when it runs. |

### Type: `dependency-mapping`

| Key                   | Value   | Description                                                                                       |
| --------------------- | ------- | ------------------------------------------------------------------------------------------------- |
| `<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>` |

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0

