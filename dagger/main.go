package main

import (
	"context"
)

type PluginCi struct{}

var php = []string{"7.4", "8.0", "8.1"}
var moodle_version = []string{"MOODLE_401_STABLE"}
var database = []string{"pgsql", "mariadb"}

func (m *PluginCi) Init(ctx context.Context) *Container {
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

	postgres := dag.Container().
		From("postgres:13").
		WithEnvVariable("POSTGRES_USER", "postgres").
		WithEnvVariable("POSTGRES_HOST_AUTH_METHOD", "trust").
		WithExposedPort(5432).
		AsService()

	// Setup moodle

	return dag.Container().
		From("php:8.1-fpm-bullseye").
		WithServiceBinding("mariadb", mariadb).
		WithServiceBinding("pgsql", postgres).
		WithExec([]string{"echo", "max_input_vars=5000", ">>", "/usr/local/etc/php/php.ini-production"})
	// Begin running tests

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
