package secrets

import (
	"fmt"

	"github.com/10gen/realm-cli/internal/cli"
	"github.com/10gen/realm-cli/internal/cli/user"
	"github.com/10gen/realm-cli/internal/cloud/realm"
	"github.com/10gen/realm-cli/internal/terminal"

	"github.com/spf13/pflag"
)

// CommandMetaList is the command meta for the `secrets list` command
var CommandMetaList = cli.CommandMeta{
	Use:         "list",
	Aliases:     []string{"ls"},
	Display:     "secrets list",
	Description: "List the Secrets in your Realm app",
	HelpText:    `This will display the IDs and Names of the Secrets in your Realm app.`,
}

// CommandList is the `secrets list` command
type CommandList struct {
	inputs listInputs
}

type listInputs struct {
	cli.ProjectInputs
}

// Flags are the command flags
func (cmd *CommandList) Flags(fs *pflag.FlagSet) {
	cmd.inputs.Flags(fs)
}

// Inputs are the command inputs
func (cmd *CommandList) Inputs() cli.InputResolver {
	return &cmd.inputs
}

// Handler is the command handler
func (cmd *CommandList) Handler(profile *user.Profile, ui terminal.UI, clients cli.Clients) error {
	app, appErr := cli.ResolveApp(ui, clients.Realm, cmd.inputs.Filter())
	if appErr != nil {
		return appErr
	}

	secrets, secretsErr := clients.Realm.Secrets(app.GroupID, app.ID)
	if secretsErr != nil {
		return secretsErr
	}

	if len(secrets) == 0 {
		ui.Print(terminal.NewTextLog("No available secrets to show"))
		return nil
	}

	ui.Print(terminal.NewTableLog(
		fmt.Sprintf("Found %d secrets", len(secrets)),
		tableHeaders(),
		tableRowsList(secrets)...,
	))
	return nil
}

func tableRowsList(secrets []realm.Secret) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0, len(secrets))
	for _, secret := range secrets {
		rows = append(rows, map[string]interface{}{
			headerName: secret.Name,
			headerID:   secret.ID,
		})
	}
	return rows
}

func (i *listInputs) Resolve(profile *user.Profile, ui terminal.UI) error {
	if err := i.ProjectInputs.Resolve(ui, profile.WorkingDirectory, false); err != nil {
		return err
	}

	return nil
}
