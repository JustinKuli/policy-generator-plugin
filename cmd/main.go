package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/pflag"
	"open-cluster-management.io/ocm-kustomize-generator-plugins/internal"
	"open-cluster-management.io/ocm-kustomize-generator-plugins/internal/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

var debug = false

func main() {
	// Parse command input
	debugFlag := pflag.Bool("debug", false, "Print the stack trace with error messages")
	pflag.Parse()

	debug = *debugFlag

	// Collect and parse PolicyGeneratorConfig file paths
	generators := pflag.Args()
	var outputBuffer bytes.Buffer

	if len(generators) == 0 {
		runKRMplugin(os.Stdin, os.Stdout)

		return
	}

	for _, gen := range generators {
		outputBuffer.Write(processGeneratorConfig(gen))
	}

	// Output results to stdout for Kustomize to handle
	// nolint:forbidigo
	fmt.Print(outputBuffer.String())
}

// errorAndExit takes a message string with formatting verbs and associated formatting
// arguments similar to fmt.Errorf(). If `debug` is set or it is given an empty message
// string, it throws a panic to print the message along with the trace. Otherwise
// it prints the formatted message to stderr and exits with error code 1.
func errorAndExit(msg string, formatArgs ...interface{}) {
	printArgs := make([]interface{}, len(formatArgs))
	copy(printArgs, formatArgs)
	// Show trace if the debug flag is set
	if msg == "" || debug {
		panic(fmt.Sprintf(msg, printArgs...))
	}

	fmt.Fprintf(os.Stderr, msg, printArgs...)
	fmt.Fprint(os.Stderr, "\n")
	os.Exit(1)
}

func runKRMplugin(input io.Reader, output io.Writer) {
	kioreader := kio.ByteReader{Reader: input}

	inputs, err := kioreader.Read()
	if err != nil {
		errorAndExit("kioreader error: %v", err)
	}

	config, err := kioreader.FunctionConfig.MarshalJSON()
	if err != nil {
		errorAndExit("unable to marshal configuration: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		errorAndExit("failed to determine the current directory: %v", err)
	}

	p := internal.Plugin{}

	err = p.Config(config, cwd)
	if err != nil {
		errorAndExit("error processing the PolicyGenerator file: %s", err)
	}

	// in KRM generator mode, this annotation will be set by kustomize
	if inputs[0].GetAnnotations()["config.kubernetes.io/local-config"] != "true" {
		inpFile, err := os.CreateTemp(".", "transformer-intput-*.yaml")
		if err != nil {
			errorAndExit("error creating an input file: %v", err)
		}

		defer os.Remove(inpFile.Name()) // clean up

		inpwriter := kio.ByteWriter{
			Writer: inpFile,
			ClearAnnotations: []string{
				"config.k8s.io/id",
				"internal.config.kubernetes.io/annotations-migration-resource-id",
				"internal.config.kubernetes.io/id",
				"kustomize.config.k8s.io/id",
			},
		}

		err = inpwriter.Write(inputs)
		if err != nil {
			errorAndExit("error writing stdin to the input file: %v", err)
		}

		p.Policies[0].Manifests = []types.Manifest{{Path: inpFile.Name()}}
	}

	generatedOutput, err := p.Generate()
	if err != nil {
		errorAndExit("error generating policies from the PolicyGenerator file: %s", err)
	}

	// Write the result in a ResourceList
	kiowriter := kio.ByteReadWriter{
		Reader:             bytes.NewBuffer(generatedOutput),
		Writer:             output,
		WrappingAPIVersion: "config.kubernetes.io/v1",
		WrappingKind:       "ResourceList",
	}

	nodes, err := kiowriter.Read()
	if err != nil {
		errorAndExit("error reading generator output: %v", err)
	}

	err = kiowriter.Write(nodes)
	if err != nil {
		errorAndExit("error writing generator output: %v", err)
	}
}

// processGeneratorConfig takes a string file path to a PolicyGenerator YAML file.
// It reads the file, processes and validates the contents, uses the contents to
// generate policies, and returns the generated policies as a byte array.
func processGeneratorConfig(filePath string) []byte {
	cwd, err := os.Getwd()
	if err != nil {
		errorAndExit("failed to determine the current directory: %v", err)
	}

	p := internal.Plugin{}

	// #nosec G304
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		errorAndExit("failed to read file '%s': %s", filePath, err)
	}

	err = p.Config(fileData, cwd)
	if err != nil {
		errorAndExit("error processing the PolicyGenerator file '%s': %s", filePath, err)
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		errorAndExit("failed to read stdin: %v", err)
	}

	if fi.Size() != 0 {
		// Running as a transformer: use stdin as the only manifest
		p.Policies[0].Manifests = []types.Manifest{{Path: "stdin"}}
	}

	generatedOutput, err := p.Generate()
	if err != nil {
		errorAndExit("error generating policies from the PolicyGenerator file '%s': %s", filePath, err)
	}

	return generatedOutput
}
