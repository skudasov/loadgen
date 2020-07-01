package main

import (
	"flag"
	"github.com/skudasov/loadgen"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	genCfgPath := flag.String("gen_config", "generator.yaml", "generator config filepath")
	flag.Parse()
	if *genCfgPath == "" {
		log.Fatal("provide path to generator config, -gen_config example.yaml")
	}
	cfg := loadgen.LoadDefaultGeneratorConfig(*genCfgPath)

	app := &cli.App{
		Name: "loadcli",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "gen_config",
				DefaultText: "generator.yaml",
				Usage:       "generator config filepath",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "cluster",
				Aliases: []string{"c"},
				Usage:   "creates load test cluster infrastructure using various providers",
				Action: func(c *cli.Context) error {
					platform := c.Args().Get(0)
					if platform != "linux" && platform != "darwin" {
						log.Fatal("platform must be one of: linux|darwin")
					}
					loadgen.BuildSuiteCommand(cfg.LoadScriptsDir, platform)
					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name:    "new",
						Aliases: []string{"n"},
						Usage:   "create new cluster on aws",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:        "nodes",
								DefaultText: "1",
								Value:       1,
								Usage:       "amount of cluster nodes",
							},
							&cli.StringFlag{
								Name:        "region",
								DefaultText: "",
								Usage:       "aws region",
								Required:    true,
							},
							&cli.StringFlag{
								Name:        "instance_type",
								DefaultText: "t2.micro",
								Value:       "t2.micro",
								Usage:       "aws region",
							},
						},
						Action: func(c *cli.Context) error {
							nodes := c.Int("nodes")
							region := c.String("region")
							inst := c.String("instance_type")
							log.Printf("creating cluster with %d (%s) nodes in aws region: %s", nodes, inst, region)
							spec := loadgen.CreateSpec(region, nodes, inst)
							ip := loadgen.NewInfrastructureProviderAWS(spec)
							ip.Bootstrap()
							return nil
						},
					},
				},
			},
			{
				Name:    "build",
				Aliases: []string{"b"},
				Usage:   "build load test for specified platform",
				Action: func(c *cli.Context) error {
					platform := c.Args().Get(0)
					if platform != "linux" && platform != "darwin" {
						log.Fatal("platform must be one of: linux|darwin")
					}
					loadgen.BuildSuiteCommand(cfg.LoadScriptsDir, platform)
					return nil
				},
			},
			{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "generates code for load test",
				Action: func(c *cli.Context) error {
					label := c.Args().Get(0)
					if label == "" {
						log.Fatalf("label must not be empty, prefer snake_case labels")
					}
					loadgen.CodegenMainFile(cfg.LoadScriptsDir)
					loadgen.GenerateNewTestCommand(cfg.LoadScriptsDir, label)
					return nil
				},
			},
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "run load test suite",
				Action: func(c *cli.Context) error {
					suiteCfg := c.Args().Get(0)
					if suiteCfg == "" {
						log.Fatal("path to load suite config must be specified")
					}
					loadgen.RunSuiteCommand(suiteCfg)
					return nil
				},
			},
			{
				Name:    "dashboard",
				Aliases: []string{"d"},
				Usage:   "regenerate & upload grafana dashboard",
				Action: func(c *cli.Context) error {
					loadgen.UploadGrafanaDashboard()
					return nil
				},
			},
			{
				Name:    "upload",
				Aliases: []string{"u"},
				Usage:   "upload load test to remote vm",
				Flags:   []cli.Flag{},
				Action: func(c *cli.Context) error {
					loadgen.BuildSuiteCommand(cfg.LoadScriptsDir, "linux")
					if len(cfg.Remotes) == 0 {
						log.Fatal("no targets to upload, please define remote section in config")
					}
					keyPathArg := c.Args().Get(0)
					for _, r := range cfg.Remotes {
						var keyPath string
						if r.KeyPath == "" && keyPathArg == "" {
							log.Printf("no key file found for: %s", r.Name)
							continue
						}
						if keyPathArg != "" {
							keyPath = keyPathArg
						} else {
							keyPath = "~/.ssh/id_rsa"
						}
						loadgen.UploadSuiteCommand(cfg.LoadScriptsDir, r.RemoteRootDir, keyPath)
					}
					return nil
				},
			},
			{
				Name:    "scaling_report",
				Aliases: []string{"sr"},
				Usage:   "plot scaling report",
				Flags:   []cli.Flag{},
				Action: func(c *cli.Context) error {
					inputCSV := c.Args().Get(0)
					outputPNG := c.Args().Get(1)
					if inputCSV == "" || outputPNG == "" {
						log.Fatal("usage: provide scaling csv file, and png name, ex: scaling.csv report.png")
					}
					loadgen.ReportScaling(inputCSV, outputPNG)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
