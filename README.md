# `gcr.io/paketo-buildpacks/gradle`

The Paketo Buildpack for Gradle is a Cloud Native Buildpack that build Gradle-based applications from source.

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
* Removes the source code in `<APPLICATION_ROOT>`, following include/exclude rules
* If `$BP_GRADLE_BUILT_ARTIFACT` matched a single file
  * Restores `$BP_GRADLE_BUILT_ARTIFACT` from the layer, expands the single file to `<APPLICATION_ROOT>`
* If `$BP_GRADLE_BUILT_ARTIFACT` matched a directory or multiple files
  * Restores the files matched by `$BP_GRADLE_BUILT_ARTIFACT` to `<APPLICATION_ROOT>`
* If `$BP_JAVA_INSTALL_NODE` is set to true and the buildpack finds one of the following at `<APPLICATION_ROOT>` or at the path set by `$BP_NODE_PROJECT_PATH`:
  * a `yarn.lock` file, the buildpack requests that `yarn` and `node` are installed at build time
  * a `package.json` file, the buildpack requests that `node` is installed at build time

## Configuration

| Environment Variable                    | Description                                                                                                                                                                                                                                                                                                                                                          |
|-----------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `$BP_GRADLE_BUILD_ARGUMENTS`            | Configure the arguments to pass to build system. Defaults to `--no-daemon -Dorg.gradle.welcome=never assemble`.                                                                                                                                                                                                                                                                                 |
| `$BP_GRADLE_ADDITIONAL_BUILD_ARGUMENTS` | Configure the additional arguments to pass to build system. Defaults to empty string.                                                                                                                                                                                                                                                          |
| `$BP_GRADLE_BUILD_FILE`                 | Configure the location of the build configuration file. If it doesn't exist this build pack will not be applied. Defaults to `build.gradle`.                                                                                                                                                                                                                         |
| `$BP_GRADLE_BUILT_MODULE`               | Configure the module to find application artifact in. Defaults to the root module (empty).                                                                                                                                                                                                                                                                           |
| `$BP_GRADLE_BUILT_ARTIFACT`             | Configure the built application artifact explicitly. Supersedes `$BP_GRADLE_BUILT_ARTIFACT`. Defaults to `build/libs/*.[jw]ar`. Can match a single file, multiple files or a directory. Can be one or more space separated patterns.                                                                                                                                 |
| `$BP_GRADLE_INIT_SCRIPT_PATH`           | Specifies a custom location to a Gradle init script, i.e. a `init.gradle` file.                                                                                                                                                                                                                                                                                      |
| `$BP_INCLUDE_FILES`                     | Colon separated list of glob patterns to match source files. Any matched file will be retained in the final image. Defaults to `` (i.e. nothing).                                                                                                                                                                                                                    |
| `$BP_EXCLUDE_FILES`                     | Colon separated list of glob patterns to match source files. Any matched file will be specifically removed from the final image. If include patterns are also specified, then they are applied first and exclude patterns can be used to further reduce the fileset.                                                                                                 |
| `$BP_JAVA_INSTALL_NODE`                 | Configure whether to request that `yarn` and `node` are installed by another buildpack**. If set to `true`, the buildpack will check the app root or path set by `$BP_NODE_PROJECT_PATH` for either: A `yarn.lock` file, which requires that `yarn` and `node` are installed or, a `package.json` file, which requires that `node` is installed. Defaults to `false` |
| `$BP_NODE_PROJECT_PATH`                 | Configure a project subdirectory to look for `package.json` and `yarn.lock` files                                                                                                                                                                                                                                                                                    |

### Note
** If the node and/or yarn requirements are met and the [Node Engine](https://github.com/paketo-buildpacks/node-engine) or [Yarn](https://github.com/paketo-buildpacks/yarn) participate in the build, environment variables related to these buildpacks can be set, such as `BP_NODE_PROJECT_PATH` or `BP_NODE_VERSION`. See the [Paketo Node.js docs](https://paketo.io/docs/howto/nodejs/) for more info.

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
