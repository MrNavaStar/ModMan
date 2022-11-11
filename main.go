package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/mrnavastar/modman/api"
	"github.com/mrnavastar/modman/services"
	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

type failedMod struct {
	UserIn string
	Slug   string
}

func main() {
	app := &cli.App{
		Name:  "ModMan",
		Usage: "Manage your mods with ease",
		Commands: []*cli.Command{
			{
				Name:        "init",
				Usage:       "init",
				Description: "Setup modman on your system",
				Action: func(c *cli.Context) error {
					pterm.DefaultCenter.Println(pterm.FgLightCyan.Sprint(figure.NewFigure("ModMan", "speed", true)))

					pterm.DefaultCenter.Print(pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgGreen)).WithMargin(10).Sprint("V" + util.GetVersion() + " - Created By MrNavaStar"))
					pterm.DefaultCenter.WithCenterEachLineSeparately().Println("Welcome!\nHelp contribute to this project over at:\nGit: https://github.com/MrNavaStar/ModMan\nIssues: https://github.com/MrNavaStar/ModMan/issues")

					reader := bufio.NewReader(os.Stdin)
					pterm.Info.Println("Enter the path to your .minecraft folder:")
					pterm.FgDarkGray.Print(">>> ")
					workDir, _ := reader.ReadString('\n')
					workDir = strings.ReplaceAll(workDir, "\n", "")

					fileutils.Setup(workDir)
					pterm.Success.Println("Setup complete")
					return nil
				},
			},
			{
				Name:        "ls",
				Aliases:     []string{"list"},
				Usage:       "ls",
				Description: "List all instances",
				Action: func(c *cli.Context) error {
					state := fileutils.LoadAppState()

					if len(state.Instances) == 0 {
						return nil
					}

					fmt.Println()
					var instances [][]string
					instances = append(instances, []string{" ", "Name", "Version", "Mods"})
					for _, instance := range state.Instances {
						var prefix string
						if state.ActiveInstance == instance.Name {
							prefix = ">"
						}

						instances = append(instances, []string{prefix, instance.Name, instance.Version, fmt.Sprint(len(instance.Mods))})
					}
					pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData(instances)).Render()
					fmt.Println()
					return nil
				},
			},
			{
				Name:        "make",
				Usage:       "make [name] [mc version]",
				Description: "Create a new instance",
				Action: func(c *cli.Context) error {
					name := c.Args().Get(0)
					version := c.Args().Get(1)

					if version == "" {
						version = api.GetLatestMcVersion()
					}

					if !api.IsVersionSupported(version) {
						pterm.Error.Println("Version not supported ~ Lowest supported is 18w43b (1.14)")
						return nil
					}

					pterm.Info.Println("Creating " + name)
					err1 := services.CreateInstance(name, version)
					if err1 != nil {
						pterm.Error.Println("Instance with that name already exists")
						return nil
					}

					pterm.Success.Println("Created " + name)
					services.SetActiveInstance(name)
					return nil
				},
			},
			{
				Name:        "sel",
				Aliases:     []string{"select"},
				Usage:       "sel [instance name]",
				Description: "Select an instance",
				Action: func(c *cli.Context) error {
					instance, err := services.GetInstance(c.Args().Get(0))
					if err != nil {
						pterm.Error.Println("No instance with that name")
						return nil
					}

					pterm.Info.Println("Now modifying " + instance.Name)
					services.SetActiveInstance(instance.Name)
					return nil
				},
			},
			{
				Name:        "rm",
				Aliases:     []string{"remove"},
				Usage:       "rm [instance name]",
				Description: "Remove an instance",
				Action: func(c *cli.Context) error {
					args := c.Args()
					_, err := services.GetInstance(args.Get(0))
					if err != nil {
						pterm.Error.Println("Failed to find an instance with that name")
						return nil
					}

					reader := bufio.NewReader(os.Stdin)
					pterm.Info.Print("Are you sure? [y/N]: ")
					input, _ := reader.ReadString('\n')
					input = strings.Replace(input, "\n", "", -1)

					if input == "y" || input == "Y" {
						services.DeleteInstance(args.Get(0))
						pterm.Success.Println("Removed " + args.Get(0))
					} else {
						pterm.Warning.Println("Action canceled")
					}
					return nil
				},
			},
			{
				Name:        "install",
				Usage:       "install [mod slug 1] [mod slug 2] [mod slug 3]",
				Description: "Install mods - as many as you like. Curseforge slugs marked with c: at the start. Ex: c:sodium",
				Action: func(c *cli.Context) error {
					args := c.Args()
					state := fileutils.LoadAppState()

					instance, err1 := services.GetInstance(state.ActiveInstance)
					if err1 != nil {
						pterm.Error.Println("Must select an instance to modify ~ modman sel <name>")
						return nil
					}

					var retrymods []failedMod
					mods := args.Slice()
					prefix := ""
					for i := 0; i < len(mods); i++ {
						mod := mods[i]

						err2 := services.AddMod(&instance, prefix+mod, util.ModData{}, false)
						if err2 != nil {
							if err2.Error() == "mod already added" {
								pterm.Info.Println(mod + " has already been added")
								continue
							}

							if err2.Error() == "invalid slug" {
								slug, err3 := api.SearchModrinth(mod)
								if err3 != nil {
									pterm.Error.Println("Could not find mod under " + mod)
									continue
								}

								var failedMod failedMod
								failedMod.UserIn = mod
								failedMod.Slug = slug
								retrymods = append(retrymods, failedMod)
								continue
							}

							if err2.Error() == "failed to find matching version" {
								pterm.Error.Println(mod + " does not have a release for " + instance.Version)
							}
						}
					}

					for _, mod := range retrymods {
						pterm.Error.Println("Failed to find mod under " + mod.UserIn)
						reader := bufio.NewReader(os.Stdin)
						pterm.Info.Print("Would you like to try under " + mod.Slug + "? [Y/n]: ")
						input, _ := reader.ReadString('\n')
						input = strings.Replace(input, "\n", "", -1)

						if strings.EqualFold(input, "y") || input == "" {
							err3 := services.AddMod(&instance, mod.Slug, util.ModData{}, false)
							if err3 != nil {
								if err3.Error() == "mod already added" {
									pterm.Error.Println(mod.Slug + " has already been added")
									continue
								}
								pterm.Error.Println(err3)
							}
						}
					}
					return services.SaveInstance(instance)
				},
			},
			{
				Name:        "uninstall",
				Usage:       "uninstall [mod slug 1] [mod slug 2] [mod slug 3]",
				Description: "Uninstall mods - as many as you like. Do not use c:",
				Action: func(c *cli.Context) error {
					args := c.Args()
					state := fileutils.LoadAppState()

					instance, err1 := services.GetInstance(state.ActiveInstance)
					if err1 != nil {
						pterm.Error.Println("Must select an instance to modify ~ modman sel <name>")
						return nil
					}

					for _, mod := range args.Slice() {
						for _, modData := range instance.Mods {
							if strings.EqualFold(modData.Name, mod) || strings.EqualFold(modData.Slug, mod) || modData.ProjectId == mod {
								mods := services.GetModsRelyOn(&instance, modData.Slug)

								if len(mods) != 0 {
									message := modData.Name + " is a dependency for:"
									for i, m := range mods {
										if i == 0 {
											message += " " + m
										} else {
											message += ", " + m
										}
									}

									pterm.Warning.Println(message)
									reader := bufio.NewReader(os.Stdin)
									pterm.Info.Print("Remove anyway? [y/N]: ")
									input, _ := reader.ReadString('\n')
									input = strings.Replace(input, "\n", "", -1)

									if strings.EqualFold(input, "n") || input == "" {
										return nil
									}
								}

								services.RemoveMod(&instance, modData.Id)
							}
						}
					}
					return services.SaveInstance(instance)
				},
			},
			{
				Name:        "lsmod",
				Usage:       "lsmod",
				Description: "list mods installed on the selected instance",
				Action: func(c *cli.Context) error {
					state := fileutils.LoadAppState()

					instance, err1 := services.GetInstance(state.ActiveInstance)
					if err1 != nil {
						pterm.Error.Println("Must select an instance ~ modman sel <name>")
						return nil
					}

					if len(instance.Mods) == 0 {
						return nil
					}

					fmt.Println()
					var mods [][]string
					mods = append(mods, []string{"Name", "Version", "Filename"})
					for _, mod := range instance.Mods {
						mods = append(mods, []string{mod.Name, mod.Version, mod.Filename})
					}
					pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData(mods)).Render()
					fmt.Println()
					return nil
				},
			},
			{
				Name:        "update",
				Usage:       "update",
				Description: "updates the selected instance",
				Action: func(c *cli.Context) error {
					state := fileutils.LoadAppState()

					instance, err := services.GetInstance(state.ActiveInstance)
					if err != nil {
						pterm.Error.Println("Must select an instance to update ~ modman sel <name>")
						return nil
					}

					pterm.Info.Println("Updating " + instance.Name)
					services.UpdateInstance(&state, instance.Name)
					fileutils.SaveAppState(state)
					pterm.Success.Println("Update complete")
					return nil
				},
			},
			{
				Name:        "migrate",
				Usage:       "migrate [mc version]",
				Description: "migrates the selected instance to the inputed game version",
				Action: func(c *cli.Context) error {
					state := fileutils.LoadAppState()
					version := c.Args().Get(0)

					oldInstance, err := services.GetInstance(state.ActiveInstance)
					if err != nil {
						pterm.Error.Println("Must select an instance to migrate ~ modman sel <name>")
						return nil
					}

					if version == "" {
						pterm.Error.Println("Please enter a valid minecraft version")
						return nil
					}

					if !api.IsVersionSupported(version) {
						pterm.Error.Println("Version not supported ~ Lowest supported is 18w43b (1.14)")
						return nil
					}

					pterm.Info.Println("Migrating " + state.ActiveInstance + " to " + version)
					newName := state.ActiveInstance + "_Migrated"
					err1 := services.CreateInstance(newName, version)
					if err1 != nil {
						newName = oldInstance.Name + "_Migrated:" + strings.ReplaceAll(time.Now().Format(time.RFC822), " ", "_")
						services.CreateInstance(newName, version)
					}

					newInstance, _ := services.GetInstance(newName)

					for _, mod := range oldInstance.Mods {
						if len(services.GetModsRelyOn(&oldInstance, mod.Slug)) == 0 {
							err2 := services.AddMod(&newInstance, mod.ProjectId, util.ModData{}, false)
							if err2 != nil && err2.Error() == "failed to find matching version" {
								pterm.Error.Println(mod.Name + " does not have a version for " + version)
							}
						}
					}

					util.Fatal(services.SaveInstance(newInstance))
					services.SetActiveInstance(newName)
					pterm.Success.Println("Migration Complete")
					return nil
				},
			},
			{
				Name:        "rename",
				Usage:       "rename [new name]",
				Description: "Renames the selected instance",
				Action: func(c *cli.Context) error {
					state := fileutils.LoadAppState()
					instance, err := services.GetInstance(state.ActiveInstance)
					if err != nil {
						pterm.Error.Println("Must select an instance to modify ~ modman sel <name>")
						return nil
					}

					oldName := instance.Name
					instance.Name = c.Args().Get(0)

					for i, in := range state.Instances {
						if in.Name == oldName {
							state.Instances[i] = instance
							break
						}
					}

					fileutils.SaveAppState(state)
					services.SetActiveInstance(instance.Name)
					pterm.Success.Println("Renamed " + oldName + " to " + instance.Name)
					return nil
				},
			},
			{
				Name:        "export",
				Usage:       "export",
				Description: "Exports the selected instance",
				Action: func(c *cli.Context) error {
					state := fileutils.LoadAppState()
					instance, err := services.GetInstance(state.ActiveInstance)
					if err != nil {
						pterm.Error.Println("Must select an instance to modify ~ modman sel <name>")
						return nil
					}

					pterm.Info.Println("Exporting " + instance.Name)
					services.ExportInstance(instance)
					pterm.Success.Println("Exported " + instance.Name)
					return nil
				},
			},
			{
				Name:        "import",
				Usage:       "import [instance | mods] [instance.json]",
				Description: "Imports an instance from an exported json",
				Action: func(c *cli.Context) error {
					method := c.Args().Get(0)
					file := c.Args().Get(1)

					if method == "instance" {
						pterm.Info.Println("Importing " + file)
						name := services.ImportInstance(file)
						pterm.Success.Println("Imported " + name)
					}

					if method == "mods" {
						pterm.Info.Println("Importing mods from" + file)

						state := fileutils.LoadAppState()
						instance, err := services.GetInstance(state.ActiveInstance)
						if err != nil {
							pterm.Error.Println("Must select an instance to modify ~ modman sel <name>")
							return nil
						}

						services.ImportMods(&instance, file)
						pterm.Success.Println("Imported mods from " + file)
					}

					return nil
				},
			},
			{
				Name:        "v",
				Aliases:     []string{"version"},
				Usage:       "v",
				Description: "Show modman version",
				Action: func(c *cli.Context) error {
					pterm.Info.Println("v" + util.GetVersion())
					return nil
				},
			},
		},
	}
	util.Fatal(app.Run(os.Args))
}
