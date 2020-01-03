// hcltemplate is a filter program for rendering JSON input to textual output
// using the HCL template language.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	flag "github.com/spf13/pflag"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/json"
	"golang.org/x/crypto/ssh/terminal"
)

var files = make(map[string]*hcl.File)

var GitCommit string
var Version = "v0.0.0"
var Prerelease = "dev"

func main() {
	flag.Usage = usage

	// We don't have any "real" optional arguments yet, but we'll call
	// Parse here both to handle --help and to reserve the POSIX extended flag
	// syntax in case we need to use it in future.
	versionP := flag.BoolP("version", "v", false, "show version information")
	flag.Parse()

	if *versionP {
		versionStr := Version
		if Prerelease != "" {
			versionStr = versionStr + "-" + Prerelease
		}
		fmt.Printf("hcltemplate %s\n", versionStr)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	var diags hcl.Diagnostics

	tmplFn := args[0]
	tmplSrc, err := ioutil.ReadFile(tmplFn)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot read template file",
			Detail:   fmt.Sprintf("Could not read %s: %s.", tmplFn, err),
		})
		exitWithDiags(diags)
	}

	// Record our file for diagnostic purposes. The HCL printer API expects
	// files to be *hcl.File, but since we're parsing a template we won't
	// get one of those and so we'll just synthesize one instead.
	files[tmplFn] = &hcl.File{Bytes: tmplSrc}

	expr, moreDiags := hclsyntax.ParseTemplate(tmplSrc, tmplFn, hcl.Pos{Line: 1, Column: 1})
	diags = append(diags, moreDiags...)
	if moreDiags.HasErrors() {
		exitWithDiags(diags)
	}

	jsonSrc, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot read input data",
			Detail:   fmt.Sprintf("Could not read JSON input data from stdin: %s.", err),
		})
		exitWithDiags(diags)
	}

	ty, err := json.ImpliedType(jsonSrc)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot read input data",
			Detail:   fmt.Sprintf("Could not read JSON input data from stdin: %s.", err),
		})
		exitWithDiags(diags)
	}

	val, err := json.Unmarshal(jsonSrc, ty)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot read input data",
			Detail:   fmt.Sprintf("Could not read JSON input data from stdin: %s.", err),
		})
		exitWithDiags(diags)
	}

	if !val.Type().IsObjectType() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid input data",
			Detail:   "Input data on stdin must be a JSON object.",
		})
		exitWithDiags(diags)
	}

	ctx := &hcl.EvalContext{
		Variables: val.AsValueMap(),
		Functions: functions(),
	}
	resultVal, moreDiags := expr.Value(ctx)
	diags = append(diags, moreDiags...)
	if moreDiags.HasErrors() {
		exitWithDiags(diags)
	}

	resultStrVal, err := convert.Convert(resultVal, cty.String)
	if err != nil || !resultStrVal.Type().Equals(cty.String) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid template result",
			Detail:   "Template must produce a string result.",
		})
		exitWithDiags(diags)
	}

	fmt.Print(resultStrVal.AsString())
}

func showDiags(diags hcl.Diagnostics) {
	color := true
	w, _, err := terminal.GetSize(1)
	if err != nil {
		w = 72
		color = false // assume failure to read size means not a terminal
	}

	pr := hcl.NewDiagnosticTextWriter(os.Stderr, files, uint(w), color)
	err = pr.WriteDiagnostics(diags)
	if err != nil {
		// If we already can't write to stderr then this is unlikely to do
		// us any good, but we'll try it anyway.
		log.Fatalf("failed to write diagnostic messages: %s", err)
	}
}

func exitWithDiags(diags hcl.Diagnostics) {
	showDiags(diags)
	if diags.HasErrors() {
		os.Exit(1)
	}
	os.Exit(0)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: hcltemplate <templatefile>\n\nThis program expects to find valid JSON object data on its stdin, which it will use to render the given template.\n\n")
}

func functions() map[string]function.Function {
	return map[string]function.Function{
		"abs":        stdlib.AbsoluteFunc,
		"can":        tryfunc.CanFunc,
		"csvdecode":  stdlib.CSVDecodeFunc,
		"coalesce":   stdlib.CoalesceFunc,
		"concat":     stdlib.ConcatFunc,
		"convert":    typeexpr.ConvertFunc,
		"format":     stdlib.FormatFunc,
		"formatdate": stdlib.FormatDateFunc,
		"int":        stdlib.IntFunc,
		"jsondecode": stdlib.JSONDecodeFunc,
		"jsonencode": stdlib.JSONEncodeFunc,
		"length":     stdlib.LengthFunc,
		"lower":      stdlib.LowerFunc,
		"max":        stdlib.MaxFunc,
		"min":        stdlib.MinFunc,
		"range":      stdlib.RangeFunc,
		"regex":      stdlib.RegexFunc,
		"regexall":   stdlib.RegexAllFunc,
		"reverse":    stdlib.ReverseFunc,
		"strlen":     stdlib.StrlenFunc,
		"substr":     stdlib.SubstrFunc,
		"try":        tryfunc.TryFunc,
		"upper":      stdlib.UpperFunc,
	}
}
