package app

import (
	"os"
	"strings"

	"github.com/10gen/realm-cli/internal/cli"
	"github.com/10gen/realm-cli/internal/cli/user"
	"github.com/10gen/realm-cli/internal/local"
	"github.com/10gen/realm-cli/internal/terminal"

	"github.com/spf13/pflag"
)

// CommandMetaDiff is the command meta
var CommandMetaDiff = cli.CommandMeta{
	Use:         "diff",
	Aliases:     []string{},
	Display:     "app diff",
	Description: "Show differences between your local directory and your Realm app",
	HelpText: `Displays file-by-file differences between your local directory and the latest
version of your Realm app. If you have more than one Realm app, you will be
prompted to select a Realm app to view.`,
}

// CommandDiff is the `app diff` command
type CommandDiff struct {
	inputs diffInputs
}

type diffInputs struct {
	cli.ProjectInputs
	LocalPath           string
	IncludeDependencies bool
	IncludeHosting      bool
}

const (
	flagLocalPathDiff            = "local"
	flagLocalPathDiffUsage       = "the local path to your Realm app"
	flagIncludeDependencies      = "include-dependencies"
	flagIncludeDependenciesShort = "d"
	flagIncludeDependenciesUsage = "include to diff Realm app dependencies changes as well"
	flagIncludeHosting           = "include-hosting"
	flagIncludeHostingShort      = "s"
	flagIncludeHostingUsage      = "include to diff Realm app hosting changes as well"
)

// Flags is the command flags
func (cmd *CommandDiff) Flags(fs *pflag.FlagSet) {
	cmd.inputs.Flags(fs)

	fs.StringVar(&cmd.inputs.LocalPath, flagLocalPathDiff, "", flagLocalPathDiffUsage)
	fs.BoolVarP(&cmd.inputs.IncludeDependencies, flagIncludeDependencies, flagIncludeDependenciesShort, false, flagIncludeDependenciesUsage)
	fs.BoolVarP(&cmd.inputs.IncludeHosting, flagIncludeHosting, flagIncludeHostingShort, false, flagIncludeHostingUsage)
}

// Inputs is the command inputs
func (cmd *CommandDiff) Inputs() cli.InputResolver {
	return &cmd.inputs
}

// Handler is the command handler
func (cmd *CommandDiff) Handler(profile *user.Profile, ui terminal.UI, clients cli.Clients) error {
	app, err := local.LoadApp(cmd.inputs.LocalPath)
	if err != nil {
		return err
	}

	appToDiff, err := cli.ResolveApp(ui, clients.Realm, cmd.inputs.Filter())
	if err != nil {
		return err
	}

	diffs, err := clients.Realm.Diff(appToDiff.GroupID, appToDiff.ID, app.AppData)
	if err != nil {
		return err
	}

	if cmd.inputs.IncludeDependencies {
		uploadPath, err := local.PrepareDependencies(app, ui)
		if err != nil {
			return err
		}
		defer os.Remove(uploadPath) //nolint:errcheck

		dependenciesDiff, err := clients.Realm.DiffDependencies(appToDiff.GroupID, appToDiff.ID, uploadPath)
		if err != nil {
			return err
		}
		diffs = append(diffs, dependenciesDiff.Strings()...)
	}

	if cmd.inputs.IncludeHosting {
		hosting, err := local.FindAppHosting(app.RootDir)
		if err != nil {
			return err
		}

		appAssets, err := clients.Realm.HostingAssets(appToDiff.GroupID, appToDiff.ID)
		if err != nil {
			return err
		}

		hostingDiffs, err := hosting.Diffs(profile.HostingAssetCachePath(), appToDiff.ID, appAssets)
		if err != nil {
			return err
		}

		diffs = append(diffs, hostingDiffs.Strings()...)
	}

	if len(diffs) == 0 {
		// there are no diffs
		ui.Print(terminal.NewTextLog("Deployed app is identical to proposed version"))
		return nil
	}

	ui.Print(terminal.NewTextLog(
		"The following reflects the proposed changes to your Realm app\n%s",
		strings.Join(diffs, "\n"),
	))

	return nil
}

func (i *diffInputs) Resolve(profile *user.Profile, ui terminal.UI) error {
	if err := i.ProjectInputs.Resolve(ui, profile.WorkingDirectory, true); err != nil {
		return err
	}

	if i.LocalPath == "" {
		i.LocalPath = profile.WorkingDirectory
	}
	return nil
}
