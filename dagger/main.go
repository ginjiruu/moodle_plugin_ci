package main

import (
	"context"
	"fmt"
)

type PluginCi struct{}

var php = []string{"7.4", "8.0", "8.1", "8.2"}
var moodle_version = []string{"MOODLE_401_STABLE", "MOODLE_404_STABLE"}
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

func (m *PluginCi) PostgresService() *Service {
	return dag.Container().
		From("postgres:13").
		WithEnvVariable("POSTGRES_USER", "postgres").
		WithEnvVariable("POSTGRES_HOST_AUTH_METHOD", "trust").
		WithExposedPort(5432).
		AsService()
}

// Init function for setting up template that other jobs draw from
func (m *PluginCi) Moodle(ctx context.Context, plugin *Directory, dependencies *Directory, phpVersion string, moodleVersion string, database string) *Container {

	moodle := dag.Container().
		From(fmt.Sprintf("php:%s-apache-bullseye", phpVersion)).
		//WithServiceBinding("pgsql", postgres).
		WithExec([]string{"echo", "max_input_vars=5000", ">>", "/usr/local/etc/php/php.ini-production"}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "--yes", "git-core", "zip", "curl", "mariadb-client", "libpng-dev", "zlib1g-dev", "libicu-dev", "postgresql-client", "libzip-dev", "libxml2-dev", "libpq-dev"}).
		WithExec([]string{"docker-php-ext-install", "pdo", "pdo_mysql", "mysqli", "gd", "intl", "zip", "soap", "pgsql"}).
		WithExec([]string{"sh", "-c", "curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer"}).
		WithExec([]string{"curl", "-fsSL", "https://deb.nodesource.com/setup_20.x", "-o", "install.sh"}).
		WithExec([]string{"bash", "install.sh"}).
		WithExec([]string{"apt-get", "install", "-y", "nodejs"}).
		WithDirectory("/var/www/html", plugin.WithoutDirectory("dagger"), ContainerWithDirectoryOpts{
			Owner: "www-data:www-data",
		}).
		WithDirectory("/var/www/dependencies", dependencies.WithoutDirectory("dagger"), ContainerWithDirectoryOpts{
			Owner: "www-data:www-data",
		}).
		WithWorkdir("/var/www/html").
		WithExec([]string{"chmod", "-R", "775", "/var/www"}).
		WithEnvVariable("MOODLE_BRANCH", moodleVersion).
		WithEnvVariable("COMPOSER_ALLOW_SUPERUSER", "1").
		WithEnvVariable("DB", database).
		WithExec([]string{"composer", "create-project", "-n", "--no-dev", "--prefer-dist", "moodlehq/moodle-plugin-ci", "../ci", "^4"})

	if database == "pgsql" {
		return moodle.WithServiceBinding(
			"db",
			m.PostgresService(),
		).
			WithNewFile("/usr/local/etc/php/conf.d/docker-php-ext-additional.ini", ContainerWithNewFileOpts{
				Contents: "max_input_vars = 5000",
			}).
			WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init"}).
			WithMountedCache("/root/.composer", dag.CacheVolume("composer-cache")).
			WithMountedCache("/var/www/vendor", dag.CacheVolume("composer-vendor-cache"))
	} else if database == "mariadb" {
		return moodle.WithServiceBinding(
			"db",
			m.MariadbService(),
		).
			WithNewFile("/usr/local/etc/php/conf.d/docker-php-ext-additional.ini", ContainerWithNewFileOpts{
				Contents: "max_input_vars = 5000",
			}).
			WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init"}).
			WithMountedCache("/root/.composer", dag.CacheVolume("composer-cache")).
			WithMountedCache("/var/www/vendor", dag.CacheVolume("composer-vendor-cache"))
	} else {
		return moodle.WithNewFile("/usr/local/etc/php/conf.d/docker-php-ext-additional.ini", ContainerWithNewFileOpts{
			Contents: "max_input_vars = 5000",
		}).
			WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init"}).
			WithMountedCache("/root/.composer", dag.CacheVolume("composer-cache")).
			WithMountedCache("/var/www/vendor", dag.CacheVolume("composer-vendor-cache"))
	}

}

// run Phplint
func (m *PluginCi) PhpLint(ctx context.Context, plugin *Directory, dependencies *Directory) (string, error) {
	return m.Moodle(ctx, plugin, dependencies, "8.1", "MOODLE_401_STABLE", "pgsql").
		WithExec([]string{"../ci/bin/moodle-plugin-ci", "phplint"}).Stdout(ctx)
}

//
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpmd"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpcs"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpdoc"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "validate"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "savepoints"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "mustache"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "grunt"}).
// WithExec([]string{"../ci/bin/moodle-plugin-ci", "phpunit"}).
