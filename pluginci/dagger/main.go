package main

import (
	"context"
	"fmt"
	"slices"
)

func New(
	// Plugin to run CI on
	plugin *Directory,
	// Path to any dependencies required for the plugin
	dependencies *Directory,
	// +optional
	// +default="8.1"
	phpVersion string,
	// +optional
	// +default="MOODLE_401_STABLE"
	moodleVersion string,
	// +optional
	// +default="mariadb"
	database string,
) *PluginCi {
	return &PluginCi{
		Plugin:        plugin,
		Dependencies:  dependencies,
		PhpVersion:    phpVersion,
		MoodleVersion: moodleVersion,
		Database:      database,
	}
}

type PluginCi struct {
	Plugin        *Directory
	Dependencies  *Directory
	PhpVersion    string
	MoodleVersion string
	Database      string
}

func (m *PluginCi) Message() string {
	return fmt.Sprintf("%s, %s!, %s#", m.Database, m.MoodleVersion, m.PhpVersion)
}

type Moodle struct {
	branch          string
	supportedPhpVer []string
}

var moodle401 = Moodle{
	branch:          "MOODLE_401_STABLE",
	supportedPhpVer: []string{"7.4", "8.0", "8.1"},
}
var moodle402 = Moodle{
	branch:          "MOODLE_402_STABLE",
	supportedPhpVer: []string{"8.0", "8.1"},
}
var moodle403 = Moodle{
	branch:          "MOODLE_403_STABLE",
	supportedPhpVer: []string{"8.0", "8.1", "8.2"},
}
var moodle404 = Moodle{
	branch:          "MOODLE_404_STABLE",
	supportedPhpVer: []string{"8.1", "8.2", "8.3"},
}
var moodle310 = Moodle{
	branch:          "MOODLE_310_STABLE",
	supportedPhpVer: []string{"7.4", "8.0"},
}

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
func (m *PluginCi) Moodle(
	ctx context.Context) *Container {
	moodle := dag.Container().
		From(fmt.Sprintf("php:%s-apache-bullseye", m.PhpVersion)).
		//WithServiceBinding("pgsql", postgres).
		WithExec([]string{"echo", "max_input_vars=5000", ">>", "/usr/local/etc/php/php.ini-production"}).
		WithExec([]string{"echo", "max_input_vars=5000", ">>", "/usr/local/etc/php/php.ini-production"}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "--yes", "git-core", "zip", "curl", "mariadb-client", "libpng-dev", "zlib1g-dev", "libicu-dev", "postgresql-client", "libzip-dev", "libxml2-dev", "libpq-dev"}).
		WithExec([]string{"docker-php-ext-install", "pdo", "pdo_mysql", "mysqli", "gd", "intl", "zip", "soap", "pgsql"}).
		WithExec([]string{"sh", "-c", "curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer"}).
		WithExec([]string{"curl", "-fsSL", "https://deb.nodesource.com/setup_20.x", "-o", "install.sh"}).
		WithExec([]string{"bash", "install.sh"}).
		WithExec([]string{"apt-get", "install", "-y", "nodejs"}).
		WithDirectory("/var/www/html", m.Plugin, ContainerWithDirectoryOpts{
			Owner: "www-data:www-data",
		}).
		WithDirectory("/var/www/dependencies", m.Dependencies, ContainerWithDirectoryOpts{
			Owner: "www-data:www-data",
		}).
		WithWorkdir("/var/www/html").
		WithExec([]string{"chmod", "-R", "775", "/var/www"}).
		WithEnvVariable("MOODLE_BRANCH", m.MoodleVersion).
		WithEnvVariable("COMPOSER_ALLOW_SUPERUSER", "1").
		WithEnvVariable("DB", m.Database).
		WithMountedCache("/root/.composer", dag.CacheVolume("composer-cache")).
		WithMountedCache("/var/www/vendor", dag.CacheVolume("composer-vendor-cache")).
		WithExec([]string{"composer", "create-project", "-n", "--no-dev", "--prefer-dist", "moodlehq/moodle-plugin-ci", "../ci", "^4"}).
		WithNewFile("/usr/local/etc/php/conf.d/docker-php-ext-additional.ini", ContainerWithNewFileOpts{
			Contents: "max_input_vars = 5000",
		}).
		WithNewFile("/usr/local/etc/php/conf.d/docker-php-memlimit.ini", ContainerWithNewFileOpts{
			Contents: "memory_limit = -1",
		})
	if m.Database == "pgsql" {
		return moodle.WithServiceBinding(
			"db",
			m.PostgresService(),
		).
			WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--moodle", "/var/www/moodle", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init", "--db-name=moodle"})
	} else if m.Database == "mariadb" {
		return moodle.WithServiceBinding(
			"db",
			m.MariadbService(),
		).
			WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--moodle", "/var/www/moodle", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init", "--db-name=moodle"})
	} else {
		return moodle.WithNewFile("/usr/local/etc/php/conf.d/docker-php-ext-additional.ini", ContainerWithNewFileOpts{
			Contents: "max_input_vars = 5000",
		}).
			WithExec([]string{"../ci/bin/moodle-plugin-ci", "install", "--plugin", "./", "--extra-plugins", "/var/www/dependencies", "--db-host=db", "--no-init", "--db-name=moodle"})
	}

}

// Run specified test(s)
func (m *PluginCi) Test(
	ctx context.Context,
	// Array of testing operation to perform
	//
	// Example: dagger call --plugin=./plugin/ --dependencies=./dependencies/ test --operations="phplint,phpmd"
	//
	// Defaults to lint
	//
	// Options: phplint, phpmd, phpcs, phpdoc, validate, savepoints, mustache, grunt, phpunit, all,
	//
	// +optional
	// +default=["phplint"]
	operations []string,
) {
	if slices.Contains(operations, "all") {
		operations[0] = "phplint"
		operations = append(operations, "phpmd")
		operations = append(operations, "phpcs")
		operations = append(operations, "phpdoc")
		operations = append(operations, "validate")
		operations = append(operations, "savepoints")
		operations = append(operations, "mustache")
		operations = append(operations, "grunt")
		operations = append(operations, "phpunit")
	}
	for _, operation := range operations {
		m.Moodle(ctx).WithExec([]string{"../ci/bin/moodle-plugin-ci", operation}).Stdout(ctx)
	}
}
