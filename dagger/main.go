package main

import (
	"context"
)

type PluginCi struct{}

var php = []string{"7.4", "8.0", "8.1"}
var moodle_version = []string{"MOODLE_401_STABLE"}
var database = []string{"pgsql", "mariadb"}

func (m *PluginCi) MariadbService() *Service {
	// Initialize sidecar services
	return dag.Container().
		From("mariadb:10").
		WithEnvVariable("MYSQL_USER", "root").
		WithEnvVariable("MYSQL_ALLOW_EMPTY_PASSWORD", "true").
		WithEnvVariable("MYSQL_COLLATION_SERVER", "utf8mb4_unicode_ci").
		//WithDefaultArgs([]string{"--health-cmd=\"mysqladmin ping\"", "--health-interval", "10s", "--health-timeout", "5s", "--health-retries", "3"}).
		WithExposedPort(3306).
		AsService()
}

// Init function for setting up template that other jobs draw from
func (m *PluginCi) Init(ctx context.Context, source *Directory, dependencies *Directory) *Container {
	// postgres := dag.Container().
	// 	From("postgres:13").
	// 	WithEnvVariable("POSTGRES_USER", "postgres").
	// 	WithEnvVariable("POSTGRES_HOST_AUTH_METHOD", "trust").
	// 	WithExposedPort(5432).
	// 	AsService()

	// Setup moodle
	return dag.Container().
		From("php:8.1-apache-bookworm").
		//WithServiceBinding("pgsql", postgres).
		WithExec([]string{"echo", "max_input_vars=5000", ">>", "/usr/local/etc/php/php.ini-production"}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "--yes", "git-core", "zip", "curl", "mariadb-client", "libpng-dev", "zlib1g-dev", "libicu-dev", "libzip-dev", "libxml2-dev"}).
		WithExec([]string{"docker-php-ext-install", "pdo", "pdo_mysql", "mysqli", "gd", "intl", "zip", "soap"}).
		WithExec([]string{"sh", "-c", "curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer"}).
		WithExec([]string{"curl", "-fsSL", "https://deb.nodesource.com/setup_20.x", "-o", "install.sh"}).
		WithExec([]string{"bash", "install.sh"}).
		WithExec([]string{"apt-get", "install", "-y", "nodejs"}).
		WithDirectory("/var/www/html", source.WithoutDirectory("dagger"), ContainerWithDirectoryOpts{
			Owner: "www-data:www-data",
		}).
		WithDirectory("/var/www/dependencies", dependencies.WithoutDirectory("dagger"), ContainerWithDirectoryOpts{
			Owner: "www-data:www-data",
		}).
		WithWorkdir("/var/www/html").
		WithExec([]string{"chmod", "-R", "775", "/var/www"}).
		WithEnvVariable("MOODLE_BRANCH", moodle_version[0]).
		WithEnvVariable("COMPOSER_ALLOW_SUPERUSER", "1").
		WithEnvVariable("DB", database[1]).
		WithExec([]string{"composer", "create-project", "-n", "--no-dev", "--prefer-dist", "moodlehq/moodle-plugin-ci", "../ci", "^4"}).
		WithServiceBinding("db", m.MariadbService()).
		WithNewFile("/usr/local/etc/php/conf.d/docker-php-ext-additional.ini", ContainerWithNewFileOpts{
			Contents: "max_input_vars = 5000",
		}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "phplint"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpmd"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpcs"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpdoc"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "validate"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "savepoints"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "mustache"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "grunt"}).
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpunit"}).
		WithMountedCache("/root/.composer", dag.CacheVolume("composer-cache")).
		WithMountedCache("/var/www/vendor", dag.CacheVolume("composer-vendor-cache"))
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

// sends a query to a MariaDB service received as input and returns the response
func (m *PluginCi) UserList(ctx context.Context, svc *Service) (string, error) {
	return dag.Container().
		From("mariadb:10.11.2").
		WithServiceBinding("db", svc).
		WithExec([]string{"/usr/bin/mysql", "--user=root", "--password=secret", "--host=db", "-e", "SELECT Host, User FROM mysql.user"}).
		Stdout(ctx)
}
