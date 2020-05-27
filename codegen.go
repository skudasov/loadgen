package loadgen

import (
	"fmt"
	. "github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	attackerLabelRe        = regexp.MustCompile("(.*) = (.*)")
	runConfigsDir          = "run_configs"
	myLibPackageName       = "loadgen"
	myLibPackageImportPath = "github.com/skudasov/loadgen"
	repoPrefix             = "github.com/insolar"
	viperImportPath        = "github.com/spf13/viper"
)

type LabelKV struct {
	Label     string
	LabelName string
}

type AttackerWithLabel struct {
	Name  string
	Label string
}

func NewLabelName(label string) string {
	return strcase.ToCamel(label + "Label")
}

func NewAttackerStructName(label string) string {
	return strcase.ToCamel(label + "Attack")
}

func NewCheckFuncName(label string) string {
	return strcase.ToCamel(label + "Check")
}

// CollectLabels read all labels in labels.go
func CollectLabels(path string) []LabelKV {
	f := createFileOrOpen(fmt.Sprintf(labelsPath, path))
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	res := attackerLabelRe.FindAllString(string(data), -1)
	labels := make([]LabelKV, 0)
	for _, s := range res {
		l := strings.Split(s, "=")
		labels = append(
			labels,
			LabelKV{
				strings.Trim(l[1], " \""),
				strings.Trim(l[0], " \""),
			},
		)
	}
	return labels
}

// CodegenAttackersFile generates attacker factory code for every label it found in labels.go, struct will be camelcased with Attack suffix:
//
//package load
//
//import (
//	"github.com/skudasov/loadgen"
//	"log"
//)
//
//func AttackerFromName(name string) loadgen.Attack {
//	switch name {
//	case "user_create":
//		return loadgen.WithMonitor(new(UserCreateAttack))
///
//  ...
//
//	default:
//		log.Fatalf("unknown attacker type: %s", name)
//		return nil
//	}
//}
func CodegenAttackersFile(packageName string, labels []LabelKV) {
	cases := make([]Code, 0)
	for _, l := range labels {
		structName := NewAttackerStructName(l.Label)
		fmt.Printf("generating label: %s\ngenerating structName: %s\n", l, structName)
		cases = append(cases, Case(Lit(l.Label)).Block(
			Return(Qual(myLibPackageImportPath, "WithMonitor").Call(Id("new").Call(Id(structName)))),
		))
	}
	cases = append(cases, Default().Block(
		Qual("log", "Fatalf").Call(
			Lit("unknown attacker type: %s"),
			Id("name"),
		),
		Return(Nil()),
	))

	f := NewFile(packageName)

	imports := map[string]string{
		myLibPackageImportPath: myLibPackageName,
		"log":                  "log",
	}
	f.ImportNames(imports)

	f.Func().Id("AttackerFromName").Params(
		Id("name").Id("string"),
	).Qual(myLibPackageImportPath, "Attack").Block(
		Switch(Id("name")).Block(
			cases...,
		),
	)

	if err := f.Save(path.Join(packageName, "attackers.go")); err != nil {
		log.Fatal(err)
	}
}

func CodegenChecksFile(packageName string, labels []LabelKV) {
	cases := make([]Code, 0)
	for _, l := range labels {
		structName := NewCheckFuncName(l.Label)
		fmt.Printf("generating label: %s\ngenerating structName: %s\n", l, structName)
		cases = append(cases, Case(Lit(l.Label)).Block(
			Return(Id(structName)),
		))
	}
	cases = append(cases, Default().Block(
		Qual("log", "Fatalf").Call(
			Lit("unknown attacker type: %s"),
			Id("name"),
		),
		Return(Nil()),
	))

	f := NewFile(packageName)

	imports := map[string]string{
		myLibPackageImportPath: myLibPackageName,
		"log":                  "log",
	}
	f.ImportNames(imports)

	f.Func().Id("CheckFromName").Params(
		Id("name").Id("string"),
	).Qual(myLibPackageImportPath, "RuntimeCheckFunc").Block(
		Switch(Id("name")).Block(
			cases...,
		),
	)

	if err := f.Save(path.Join(packageName, "checks.go")); err != nil {
		log.Fatal(err)
	}
}

// CodegenLabelsFile generate labels file, add new label for created test:
//
//package load
//
//const (
//	UserCreateLabel           = "user_create"
//  ...
//)
func CodegenLabelsFile(packageName string, labels []LabelKV) {
	f := NewFile(packageName)
	statements := make([]Code, 0)
	for _, kv := range labels {
		statements = append(statements, Id(kv.LabelName).Op("=").Lit(kv.Label))
	}
	f.Const().Defs(
		statements...,
	)
	if err := f.Save(path.Join(packageName, "labels.go")); err != nil {
		log.Fatal(err)
	}
}

// CodegenAttackerFile generates load test code:
//
//package generated_loadtest
//
//import (
//	"context"
//	"github.com/skudasov/loadgen"
//)
//
//type GeneratedAttack struct {
//	loadgen.WithRunner
//}
//
//func (a *GeneratedAttack) Setup(hc loadgen.RunnerConfig) error {
//	return nil
//}
//func (a *GeneratedAttack) Do(ctx context.Context) loadgen.DoResult {
//	return loadgen.DoResult{
//		Error:        nil,
//		RequestLabel: GeneratedLabel,
//	}
//}
//func (a *GeneratedAttack) Clone(r *loadgen.Runner) loadgen.Attack {
//	return &GeneratedAttack{WithRunner: loadgen.WithRunner{R: r}}
//}
func CodegenAttackerFile(packageName string, label string) {
	structName := NewAttackerStructName(label)
	labelName := NewLabelName(label)
	f := NewFile(packageName)

	imports := map[string]string{
		myLibPackageImportPath: myLibPackageName,
		"context":              "context",
	}
	f.ImportNames(imports)

	f.Type().Id(structName).Struct(
		Qual(myLibPackageImportPath, "WithRunner"),
	)

	f.Func().Params(
		Id("a").Id("*" + structName),
	).Id("Setup").Params(
		Id("hc").Qual(myLibPackageImportPath, "RunnerConfig"),
	).Error().Block(
		Return(Nil()),
	)

	f.Func().Params(
		Id("a").Id("*" + structName),
	).Id("Do").Params(
		Id("ctx").Qual("context", "Context"),
	).Id(myLibPackageName).Dot("DoResult").Block(
		Return(
			Id(myLibPackageName).Dot("DoResult").Values(Dict{
				Id("RequestLabel"): Id(labelName),
				Id("Error"):        Nil(),
			},
			),
		),
	)

	f.Func().Params(
		Id("a").Id("*" + structName),
	).Id("Clone").Params(
		Id("r").Id("*" + myLibPackageName).Dot("Runner"),
	).Id(myLibPackageName).Dot("Attack").Block(
		Return(
			Id("&" + structName).Values(Dict{
				Id("WithRunner"): Id(myLibPackageName).Dot("WithRunner").Values(Dict{
					Id("R"): Id("r"),
				}),
			},
			),
		),
	)
	if err := f.Save(path.Join(packageName, label+"_attack.go")); err != nil {
		log.Fatal(err)
	}
}

// CodegenMainFile generate loadtest entry point:
//
//package main
//
//import (
//	"github.com/skudasov/loadgen"
//	"github.com/insolar/example_loadtest"
//	"github.com/insolar/example_loadtest/config"
//)
//
//func main() {
//	loadgen.Run(example_loadtest.AttackerFromName, example_loadtest.CheckFromName)
//}
func CodegenMainFile(packageName string) {
	targetDir := path.Join(packageName, "cmd", "load")
	createDirIfNotExists(targetDir)

	f := NewFile("main")

	rootPackageName := viper.GetString("root_package_name")
	loadTestPackageImportPath := path.Join(repoPrefix, rootPackageName, packageName)

	imports := map[string]string{
		myLibPackageImportPath:    myLibPackageName,
		loadTestPackageImportPath: packageName,
	}
	f.ImportNames(imports)

	f.Func().Id("main").Params().Block(
		Qual(myLibPackageImportPath, "Run").Call(Qual(loadTestPackageImportPath, "AttackerFromName"), Qual(loadTestPackageImportPath, "CheckFromName")),
	)

	if err := f.Save(path.Join(targetDir, "main.go")); err != nil {
		log.Fatal(err)
	}
}

// GenerateSingleRunConfig generates simple config to run test for debug
func GenerateSingleRunConfig(testDir string, label string) {
	runCfgPath := path.Join(testDir, runConfigsDir)
	createDirIfNotExists(runCfgPath)
	suiteCfg := &SuiteConfig{
		DumpTransport: true,
		HttpTimeout:   20,
		Steps: []Step{
			{
				Name:          "load",
				ExecutionMode: "sequence",
				Handles: []RunnerConfig{
					{
						HandleName:     label,
						RampUpStrategy: "exp2",
						RPS:            1,
						RampUpTimeSec:  1,
						AttackTimeSec:  30,
						MaxAttackers:   1,
						DoTimeoutSec:   40,
						Verbose:        true,
						RecycleData:    true,
					},
				},
			},
		},
	}
	cfg, err := yaml.Marshal(suiteCfg)
	if err != nil {
		log.Fatalf("failed to marshal single run config: %s", err)
	}
	if err := ioutil.WriteFile(path.Join(runCfgPath, label+".yaml"), cfg, os.ModePerm); err != nil {
		log.Fatalf("failed to write single run config for label: %s, %s", label, err)
	}
}
