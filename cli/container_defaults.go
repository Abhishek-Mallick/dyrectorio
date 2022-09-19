package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/mount"

	containerbuilder "github.com/dyrector-io/dyrectorio/agent/pkg/builder/container"
)

const PostgresImage = "docker.io/library/postgres:13-alpine"
const MailSlurperImage = "docker.io/oryd/mailslurper:latest-smtps"

// Crux services: db migrations and crux api service
func GetCrux(settings *Settings) *containerbuilder.DockerContainerBuilder {
	crux := containerbuilder.NewDockerBuilder(context.Background()).
		WithImage(fmt.Sprintf("%s:%s", settings.Crux.Image, settings.SettingsFile.Version)).
		WithName(settings.Containers.Crux.Name).
		WithRestartPolicy(containerbuilder.AlwaysRestartPolicy).
		WithoutConflict().
		WithForcePullImage().
		WithEnv([]string{
			fmt.Sprintf("TZ=%s", settings.SettingsFile.TimeZone),
			fmt.Sprintf("DATABASE_URL=postgresql://%s:%s@%s:%d/%s?schema=public",
				settings.SettingsFile.CruxPostgresUser,
				settings.SettingsFile.CruxPostgresPassword,
				settings.Containers.CruxPostgres.Name,
				DefaultPostgresPort,
				settings.SettingsFile.CruxPostgresDB),
			fmt.Sprintf("KRATOS_ADMIN_URL=http://%s:%d",
				settings.Containers.Kratos.Name,
				settings.SettingsFile.KratosAdminPort),
			fmt.Sprintf("CRUX_UI_URL=localhost:%d", settings.SettingsFile.CruxUIPort),
			fmt.Sprintf("CRUX_AGENT_ADDRESS=%s:%d", settings.NetworkGatewayIP, settings.SettingsFile.CruxAgentGrpcPort),
			"GRPC_API_INSECURE=true",
			"GRPC_AGENT_INSECURE=true",
			"GRPC_AGENT_INSTALL_SCRIPT_INSECURE=true",
			"LOCAL_DEPLOYMENT=true",
			fmt.Sprintf("LOCAL_DEPLOYMENT_NETWORK=%s", settings.SettingsFile.Network),
			fmt.Sprintf("JWT_SECRET=%s", settings.SettingsFile.CruxSecret),
			"CRUX_DOMAIN=DNS:localhost",
			"FROM_NAME=dyrector.io",
			"SENDGRID_KEY=SG.InvalidKey",
			"FROM_EMAIL=mail@szolgalta.to",
			"SMTP_USER=test",
			"SMTP_PASSWORD=test",
			fmt.Sprintf("SMTP_URL=%s:1025/?skip_ssl_verify=true&legacy_ssl=true", settings.Containers.MailSlurper.Name),
		}).
		WithPortBindings([]containerbuilder.PortBinding{
			{
				ExposedPort: DefaultCruxAgentGrpcPort,
				PortBinding: uint16(settings.SettingsFile.CruxAgentGrpcPort),
			},
			{
				ExposedPort: DefaultCruxGrpcPort,
				PortBinding: uint16(settings.SettingsFile.CruxGrpcPort),
			}}).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithNetworkAliases(settings.Containers.Crux.Name).
		WithMountPoints([]mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: fmt.Sprintf("%s-certs", settings.Containers.Crux.Name),
				Target: "/app/certs",
			},
		}).
		WithCmd([]string{"serve"})
	return crux
}

func GetCruxMigrate(settings *Settings) *containerbuilder.DockerContainerBuilder {
	cruxMigrate := containerbuilder.NewDockerBuilder(context.Background()).
		WithImage(fmt.Sprintf("%s:%s", settings.Crux.Image, settings.SettingsFile.Version)).
		WithName(settings.Containers.CruxMigrate.Name).
		WithoutConflict().
		WithForcePullImage().
		WithEnv([]string{
			fmt.Sprintf("TZ=%s", settings.SettingsFile.TimeZone),
			fmt.Sprintf("DATABASE_URL=postgresql://%s:%s@%s:%d/%s?schema=public",
				settings.SettingsFile.CruxPostgresUser,
				settings.SettingsFile.CruxPostgresPassword,
				settings.Containers.CruxPostgres.Name,
				DefaultPostgresPort,
				settings.SettingsFile.CruxPostgresDB)}).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithNetworkAliases(settings.Containers.CruxMigrate.Name).
		WithCmd([]string{"migrate"})

	return cruxMigrate
}

func GetCruxUI(settings *Settings) *containerbuilder.DockerContainerBuilder {
	cruxUI := containerbuilder.NewDockerBuilder(context.Background()).
		WithImage(fmt.Sprintf("%s:%s", settings.CruxUI.Image, settings.SettingsFile.Version)).
		WithName(settings.Containers.CruxUI.Name).
		WithRestartPolicy(containerbuilder.AlwaysRestartPolicy).
		WithoutConflict().
		WithForcePullImage().
		WithEnv([]string{
			fmt.Sprintf("TZ=%s", settings.SettingsFile.TimeZone),
			fmt.Sprintf("KRATOS_URL=http://%s:%d",
				settings.Containers.Kratos.Name,
				settings.SettingsFile.KratosPublicPort),
			fmt.Sprintf("KRATOS_ADMIN_URL=http://%s:%d",
				settings.Containers.Kratos.Name,
				settings.SettingsFile.KratosAdminPort),
			fmt.Sprintf("CRUX_API_ADDRESS=%s:%d", settings.CruxUI.CruxAddr, settings.SettingsFile.CruxGrpcPort),
			"CRUX_INSECURE=true",
			"DISABLE_RECAPTCHA=true",
		}).
		WithPortBindings([]containerbuilder.PortBinding{
			{
				ExposedPort: DefaultCruxUIPort,
				PortBinding: uint16(settings.SettingsFile.CruxUIPort),
			}}).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithNetworkAliases(settings.Containers.CruxUI.Name).
		WithMountPoints([]mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   fmt.Sprintf("%s-certs", settings.Containers.Crux.Name),
				Target:   "/app/certs",
				ReadOnly: true,
			},
		})

	return cruxUI
}

// Return Kratos services' containers
func GetKratos(settings *Settings) *containerbuilder.DockerContainerBuilder {
	kratos := containerbuilder.NewDockerBuilder(context.Background()).
		WithImage(fmt.Sprintf("%s:%s", settings.Kratos.Image, settings.SettingsFile.Version)).
		WithName(settings.Containers.Kratos.Name).
		WithRestartPolicy(containerbuilder.AlwaysRestartPolicy).
		WithoutConflict().
		WithForcePullImage().
		WithEnv([]string{
			"SQA_OPT_OUT=true",
			fmt.Sprintf("DSN=postgresql://%s:%s@%s:%d/%s?sslmode=disable&max_conns=20&max_idle_conns=4",
				settings.SettingsFile.KratosPostgresUser,
				settings.SettingsFile.KratosPostgresPassword,
				settings.Containers.KratosPostgres.Name,
				DefaultPostgresPort,
				settings.SettingsFile.KratosPostgresDB),
			fmt.Sprintf("KRATOS_URL=http://%s:%d",
				settings.Containers.Kratos.Name,
				settings.SettingsFile.KratosPublicPort),
			fmt.Sprintf("KRATOS_ADMIN_URL=http://%s:%d",
				settings.Containers.Kratos.Name,
				settings.SettingsFile.KratosAdminPort),
			fmt.Sprintf("AUTH_URL=http://%s:%d/auth", settings.Containers.CruxUI.Name, settings.SettingsFile.CruxUIPort),
			fmt.Sprintf("CRUX_UI_URL=http://%s:%d", settings.Containers.CruxUI.Name, settings.SettingsFile.CruxUIPort),
			"DEV=false",
			"LOG_LEVEL=info",
			"LOG_LEAK_SENSITIVE_VALUES=true",
			fmt.Sprintf("SECRETS_COOKIE=%s", settings.SettingsFile.KratosSecret),
			"SMTP_USER=test",
			"SMTP_PASSWORD=test",
			fmt.Sprintf("SMTP_URL=%s:1025/?skip_ssl_verify=true&legacy_ssl=true", settings.Containers.MailSlurper.Name),
			fmt.Sprintf("COURIER_SMTP_CONNECTION_URI=smtps://test:test@%s:1025/?skip_ssl_verify=true&legacy_ssl=true",
				settings.Containers.MailSlurper.Name),
		}).
		WithPortBindings([]containerbuilder.PortBinding{
			{
				ExposedPort: DefaultKratosPublicPort,
				PortBinding: uint16(settings.SettingsFile.KratosPublicPort),
			},
			{
				ExposedPort: DefaultKratosAdminPort,
				PortBinding: uint16(settings.SettingsFile.KratosAdminPort),
			}}).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithNetworkAliases(settings.Containers.Kratos.Name)

	return kratos
}

func GetKratosMigrate(settings *Settings) *containerbuilder.DockerContainerBuilder {
	kratosMigrate := containerbuilder.NewDockerBuilder(context.Background()).
		WithImage(fmt.Sprintf("%s:%s", settings.Kratos.Image, settings.SettingsFile.Version)).
		WithName(settings.Containers.KratosMigrate.Name).
		WithoutConflict().
		WithForcePullImage().
		WithEnv([]string{
			"SQA_OPT_OUT=true",
			fmt.Sprintf("DSN=postgresql://%s:%s@%s:%d/%s?sslmode=disable&max_conns=20&max_idle_conns=4",
				settings.SettingsFile.KratosPostgresUser,
				settings.SettingsFile.KratosPostgresPassword,
				settings.Containers.KratosPostgres.Name,
				DefaultPostgresPort,
				settings.SettingsFile.KratosPostgresDB),
		}).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithNetworkAliases(settings.Containers.KratosMigrate.Name).
		WithCmd([]string{"-c /etc/config/kratos/kratos.yaml", "migrate", "sql", "-e", "--yes"})

	return kratosMigrate
}

// Return Mailslurper services container
func GetMailSlurper(settings *Settings) *containerbuilder.DockerContainerBuilder {
	mailslurper := containerbuilder.NewDockerBuilder(context.Background()).
		WithImage(MailSlurperImage).
		WithName(settings.Containers.MailSlurper.Name).
		WithRestartPolicy(containerbuilder.AlwaysRestartPolicy).
		WithoutConflict().
		WithForcePullImage().
		WithPortBindings([]containerbuilder.PortBinding{
			{
				ExposedPort: DefaultMailSlurperPort,
				PortBinding: uint16(settings.SettingsFile.MailSlurperPort),
			},
			{
				ExposedPort: DefaultMailSlurperPort2,
				PortBinding: uint16(settings.SettingsFile.MailSlurperPort2),
			}}).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithNetworkAliases(settings.Containers.MailSlurper.Name)

	return mailslurper
}

// Return Postgres services' containers
func GetCruxPostgres(settings *Settings) *containerbuilder.DockerContainerBuilder {
	cruxPostgres := GetBasePostgres(settings).
		WithName(settings.Containers.CruxPostgres.Name).
		WithNetworkAliases(settings.Containers.CruxPostgres.Name).
		WithEnv([]string{
			fmt.Sprintf("POSTGRES_USER=%s", settings.SettingsFile.CruxPostgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", settings.SettingsFile.CruxPostgresPassword),
			fmt.Sprintf("POSTGRES_DB=%s", settings.SettingsFile.CruxPostgresDB),
		}).
		WithPortBindings([]containerbuilder.PortBinding{
			{
				ExposedPort: DefaultPostgresPort,
				PortBinding: uint16(settings.SettingsFile.CruxPostgresPort),
			}}).
		WithMountPoints([]mount.Mount{{
			Type:   mount.TypeVolume,
			Source: fmt.Sprintf("%s-data", settings.Containers.CruxPostgres.Name),
			Target: "/var/lib/postgresql/data"}})

	return cruxPostgres
}

func GetKratosPostgres(settings *Settings) *containerbuilder.DockerContainerBuilder {
	kratosPostgres := GetBasePostgres(settings).
		WithEnv([]string{
			fmt.Sprintf("POSTGRES_USER=%s", settings.SettingsFile.KratosPostgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", settings.SettingsFile.KratosPostgresPassword),
			fmt.Sprintf("POSTGRES_DB=%s", settings.SettingsFile.KratosPostgresDB),
		}).
		WithPortBindings([]containerbuilder.PortBinding{
			{
				ExposedPort: DefaultPostgresPort,
				PortBinding: uint16(settings.SettingsFile.KratosPostgresPort),
			}}).
		WithName(settings.Containers.KratosPostgres.Name).
		WithNetworkAliases(settings.Containers.KratosPostgres.Name).
		WithMountPoints([]mount.Mount{
			{Type: mount.TypeVolume,
				Source: fmt.Sprintf("%s-data", settings.Containers.KratosPostgres.Name),
				Target: "/var/lib/postgresql/data"}})

	return kratosPostgres
}

func GetBasePostgres(settings *Settings) *containerbuilder.DockerContainerBuilder {
	basePostgres := containerbuilder.
		NewDockerBuilder(context.Background()).
		WithImage(PostgresImage).
		WithNetworks([]string{settings.SettingsFile.Network}).
		WithRestartPolicy(containerbuilder.AlwaysRestartPolicy).
		WithoutConflict().
		WithForcePullImage()

	return basePostgres
}
