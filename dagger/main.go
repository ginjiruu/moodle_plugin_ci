package main

import (
	"context"
)

type PluginCi struct{}

func (m *PluginCi) Hello() string {
	return "Hello Moodle"
}

func (m *PluginCi) All(ctx context.Context) (string, error) {

	// Initialize sidecar services
	mariadb := dag.Container().
		From("mariadb:10").
		WithEnvVariable("MYSQL_USER", "root").
		WithEnvVariable("MARIADB_ROOT_PASSWORD", "password").
		//WithEnvVariable("MYSQL_ALLOW_EMPTY_PASSWORD", "true").
		WithEnvVariable("MYSQL_CHARACTER_SET_SERVER", "utf8mb4").
		WithEnvVariable("MYSQL_COLLATION_SERVER", "utf8mb4_unicode_ci").
		//WithDefaultArgs([]string{"--health-cmd=\"mysqladmin ping\"", "--health-interval", "10s", "--health-timeout", "5s", "--health-retries", "3"}).
		WithExposedPort(3306).
		AsService()

	// Begin running tests

	// get Drupal base image
	// install additional dependencies
	drupal := dag.Container().
		From("drupal:10.0.7-php8.2-fpm").
		WithExec([]string{"composer", "require", "drupal/core-dev", "--dev", "--update-with-all-dependencies"})

	// add service binding for MariaDB
	// run kernel tests using PHPUnit
	return drupal.
		WithServiceBinding("db", mariadb).
		WithEnvVariable("SIMPLETEST_DB", "mysql://root:password@db/drupal").
		WithEnvVariable("SYMFONY_DEPRECATIONS_HELPER", "disabled").
		WithWorkdir("/opt/drupal/web/core").
		WithExec([]string{"../../vendor/bin/phpunit", "-v", "--group", "KernelTests"}).
		Stdout(ctx)
}

// Returns a container that echoes whatever string argument is provided
func (m *PluginCi) ContainerEcho(stringArg string) *Container {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg})
}

// Returns lines that match a pattern in the files of the provided Directory
func (m *PluginCi) GrepDir(ctx context.Context, directoryArg *Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}
